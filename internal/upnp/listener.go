package upnp

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ssdpAddr   = "239.255.255.250:1900"
	apiVersion = "1.47.0"
	hubVersion = "9999999999"
	modelID    = "BSB002"
	uuidPrefix = "2f402f80-da50-11e1-9b23-"
	sendDelay  = 650 * time.Millisecond
)

type Listener struct {
	bridgeIP string
	port     int
	uuid     string
	bridgeID string
	snuuid   string
}

func NewListener(bridgeIP string, port int, uuidSuffix string) (*Listener, error) {
	if uuidSuffix == "" {
		uuidSuffix = strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	}
	mac := deriveMac(uuidSuffix)
	return &Listener{
		bridgeIP: bridgeIP,
		port:     port,
		uuid:     uuidSuffix,
		bridgeID: strings.ToUpper(mac[:6] + "FFFE" + mac[6:]),
		snuuid:   uuidSuffix,
	}, nil
}

func deriveMac(suffix string) string {
	if len(suffix) >= 12 {
		return suffix[:12]
	}
	return fmt.Sprintf("%012s", suffix)
}

func (l *Listener) Listen() {
	addr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		log.Printf("SSDP resolve error: %v", err)
		return
	}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Printf("SSDP listen error: %v", err)
		return
	}
	defer conn.Close()
	log.Printf("SSDP listener running on %s", ssdpAddr)

	buf := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("SSDP read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		msg := string(buf[:n])
		if l.isMSearch(msg) {
			go l.respond(conn, src)
		}
	}
}

func (l *Listener) isMSearch(msg string) bool {
	lower := strings.ToLower(msg)
	if !strings.HasPrefix(msg, "M-SEARCH * HTTP/1.1") {
		return false
	}
	if !strings.Contains(lower, "man: \"ssdp:discover\"") && !strings.Contains(lower, "man:\"ssdp:discover\"") {
		return false
	}
	return strings.Contains(lower, "st: urn:schemas-upnp-org:device:basic:1") ||
		strings.Contains(lower, "st: upnp:rootdevice") ||
		strings.Contains(lower, "st: ssdp:all") ||
		(strings.Contains(lower, "st: uuid:") && strings.Contains(msg, l.snuuid))
}

func (l *Listener) respond(conn *net.UDPConn, src *net.UDPAddr) {
	time.Sleep(sendDelay)
	resp := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nHOST: %s\r\nCACHE-CONTROL: max-age=100\r\nEXT:\r\n"+
			"LOCATION: http://%s:%d/description.xml\r\n"+
			"SERVER: FreeRTOS/6.0.5, UPnP/1.0, IpBridge/%s\r\n"+
			"hue-bridgeid: %s\r\n"+
			"ST: urn:schemas-upnp-org:device:basic:1\r\n"+
			"USN: uuid:%s%s\r\n\r\n",
		ssdpAddr, l.bridgeIP, l.port, apiVersion, l.bridgeID, uuidPrefix, l.snuuid,
	)
	conn.WriteToUDP([]byte(resp), src)
}

func RegisterDescription(mux *http.ServeMux, bridgeIP string, port int, uuidSuffix string) {
	if uuidSuffix == "" {
		uuidSuffix = strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	}
	mac := deriveMac(uuidSuffix)
	bridgeID := strings.ToUpper(mac[:6] + "FFFE" + mac[6:])

	desc := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" ?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
<specVersion><major>1</major><minor>0</minor></specVersion>
<URLBase>http://%s:%d/</URLBase>
<device>
  <deviceType>urn:schemas-upnp-org:device:Basic:1</deviceType>
  <friendlyName>Philips hue (%s)</friendlyName>
  <manufacturer>Royal Philips Electronics</manufacturer>
  <manufacturerURL>http://www.philips.com</manufacturerURL>
  <modelDescription>Philips hue Personal Wireless Lighting</modelDescription>
  <modelName>Philips hue bridge 2015</modelName>
  <modelNumber>%s</modelNumber>
  <modelURL>http://www.meethue.com</modelURL>
  <serialNumber>%s</serialNumber>
  <UDN>uuid:%s%s</UDN>
  <presentationURL>index.html</presentationURL>
</device>
</root>`, bridgeIP, port, bridgeIP, modelID, uuidSuffix, uuidPrefix, uuidSuffix)

	_ = bridgeID
	mux.HandleFunc("/description.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, desc)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
}
