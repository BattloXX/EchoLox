package upnp

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/BattloXX/EchoLox/internal/identity"
)

const (
	ssdpAddr   = "239.255.255.250:1900"
	apiVersion = "1.47.0"
	modelID    = "BSB002"
	uuidPrefix = "2f402f80-da50-11e1-9b23-"
)

// Listener listens for SSDP M-SEARCH packets and responds to Alexa discovery.
type Listener struct {
	info identity.BridgeInfo
}

func NewListener(info identity.BridgeInfo) *Listener {
	return &Listener{info: info}
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
	conn.SetReadBuffer(2048)
	log.Printf("SSDP listener on %s  bridgeid=%s", ssdpAddr, l.info.BridgeID)

	buf := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("SSDP read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if l.isMSearch(string(buf[:n])) {
			go l.respond(src)
		}
	}
}

func (l *Listener) isMSearch(msg string) bool {
	lower := strings.ToLower(msg)
	if !strings.HasPrefix(msg, "M-SEARCH") {
		return false
	}
	if !strings.Contains(lower, "ssdp:discover") {
		return false
	}
	return strings.Contains(lower, "st: ssdp:all") ||
		strings.Contains(lower, "st: upnp:rootdevice") ||
		// accept both Basic:1 and basic:1 (some clients send lowercase)
		strings.Contains(lower, "st: urn:schemas-upnp-org:device:basic:1") ||
		(strings.Contains(lower, "st: uuid:") && strings.Contains(msg, l.info.Suffix))
}

// respond sends a unicast 200 OK via a dedicated ephemeral socket.
// Must NOT reuse the multicast socket — the source IP would be wrong on some kernels.
func (l *Listener) respond(src *net.UDPAddr) {
	conn, err := net.DialUDP("udp4", nil, src)
	if err != nil {
		log.Printf("SSDP unicast dial: %v", err)
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(l.buildResponse())); err != nil {
		log.Printf("SSDP unicast write: %v", err)
	}
}

// buildResponse returns the exact SSDP 200 OK Alexa expects.
// USN must include ::urn:schemas-upnp-org:device:Basic:1 — without it Alexa ignores the bridge.
func (l *Listener) buildResponse() string {
	location := fmt.Sprintf("http://%s:%d/description.xml", l.info.IP, l.info.Port)
	fullUUID := uuidPrefix + l.info.Suffix
	return "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=100\r\n" +
		"EXT:\r\n" +
		"LOCATION: " + location + "\r\n" +
		"SERVER: Linux/3.14.0 UPnP/1.0 IpBridge/" + apiVersion + "\r\n" +
		"ST: urn:schemas-upnp-org:device:Basic:1\r\n" +
		"USN: uuid:" + fullUUID + "::urn:schemas-upnp-org:device:Basic:1\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		"\r\n"
}

// RegisterDescription registers /description.xml and icon stubs on the HTTP mux.
func RegisterDescription(mux *http.ServeMux, info identity.BridgeInfo) {
	desc := buildDescriptionXML(info)
	mux.HandleFunc("/description.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, desc)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/hue_logo_0.png", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/hue_logo_3.png", func(w http.ResponseWriter, r *http.Request) {})
}

// buildDescriptionXML returns the UPnP description Alexa fetches after SSDP discovery.
// serialNumber must equal bridgeid — Alexa validates this.
// UDN must match the UUID in the SSDP response.
func buildDescriptionXML(info identity.BridgeInfo) string {
	fullUUID := uuidPrefix + info.Suffix
	urlBase := fmt.Sprintf("http://%s:%d/", info.IP, info.Port)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<root xmlns="urn:schemas-upnp-org:device-1-0">
  <specVersion>
    <major>1</major>
    <minor>0</minor>
  </specVersion>
  <URLBase>%s</URLBase>
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
    <UDN>uuid:%s</UDN>
    <presentationURL>index.html</presentationURL>
    <iconList>
      <icon>
        <mimetype>image/png</mimetype>
        <height>48</height>
        <width>48</width>
        <depth>24</depth>
        <url>hue_logo_0.png</url>
      </icon>
      <icon>
        <mimetype>image/png</mimetype>
        <height>120</height>
        <width>120</width>
        <depth>24</depth>
        <url>hue_logo_3.png</url>
      </icon>
    </iconList>
  </device>
</root>`, urlBase, info.IP, modelID, info.BridgeID, fullUUID)
}
