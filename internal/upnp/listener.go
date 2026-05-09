package upnp

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/BattloXX/EchoLox/internal/discovery"
	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/logbuf"
)

var startupTime = time.Now().Unix()

const (
	ssdpAddr   = "239.255.255.250:1900"
	apiVersion = "1.16.0"
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

func (l *Listener) location() string {
	return fmt.Sprintf("http://%s:%d/description.xml", l.info.IP, l.info.Port)
}

func (l *Listener) Listen() {
	conn, err := listenMulticast(net.ParseIP(l.info.IP))
	if err != nil {
		logbuf.Global.Info("SSDP listen error: %v — Alexa discovery disabled", err)
		return
	}
	defer conn.Close()
	conn.SetReadBuffer(65536)
	logbuf.Global.Info("SSDP listener on %s  bridgeid=%s  LOCATION=%s", ssdpAddr, l.info.BridgeID, l.location())

	// Send NOTIFY announcements on start and periodically
	go func() {
		time.Sleep(2 * time.Second)
		for i := 0; i < 3; i++ {
			l.sendNotify()
			time.Sleep(200 * time.Millisecond)
		}
		ticker := time.NewTicker(120 * time.Second)
		for range ticker.C {
			for i := 0; i < 3; i++ {
				l.sendNotify()
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			logbuf.Global.Debug("SSDP read error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		msg := string(buf[:n])
		logbuf.Global.Debug("SSDP from %s: %s", src, firstLine(msg))
		if isMSearch(msg) {
			ua := extractHeader(msg, "user-agent")
			discovery.RecordAlexa(src.IP.String(), ua)
			logbuf.Global.Info("SSDP M-SEARCH from %s — responding (LOCATION: %s)", src, l.location())
			go l.respond(src, msg)
		}
	}
}

func isMSearch(msg string) bool {
	return strings.HasPrefix(msg, "M-SEARCH") && strings.Contains(strings.ToLower(msg), "ssdp:discover")
}

// respond mirrors HA's _handle_request: send rootdevice if that's what was asked,
// otherwise send the Hue device response — exactly ONE packet per M-SEARCH.
func (l *Listener) respond(src *net.UDPAddr, msg string) {
	conn, err := net.DialUDP("udp4", nil, src)
	if err != nil {
		logbuf.Global.Debug("SSDP unicast dial: %v", err)
		return
	}
	defer conn.Close()
	var resp string
	if strings.Contains(msg, "upnp:rootdevice") {
		resp = l.buildRootResponse()
	} else {
		resp = l.buildDeviceResponse()
	}
	if _, err := conn.Write([]byte(resp)); err != nil {
		logbuf.Global.Debug("SSDP unicast write: %v", err)
		return
	}
	logbuf.Global.Debug("SSDP 200 OK sent to %s", src)
}

func (l *Listener) ssdpHeader() string {
	return "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=60\r\n" +
		"EXT:\r\n" +
		"LOCATION: " + l.location() + "\r\n" +
		"SERVER: FreeRTOS/6.0.5, UPnP/1.0, IpBridge/" + apiVersion + "\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		fmt.Sprintf("BOOTID.UPNP.ORG: %d\r\n", startupTime) +
		"CONFIGID.UPNP.ORG: 1\r\n"
}

// buildRootResponse matches HA's _upnp_root_response.
func (l *Listener) buildRootResponse() string {
	fullUUID := uuidPrefix + l.info.Suffix
	return l.ssdpHeader() +
		"ST: upnp:rootdevice\r\n" +
		"USN: uuid:" + fullUUID + "::upnp:rootdevice\r\n\r\n"
}

// buildDeviceResponse matches HA's _upnp_device_response:
// lowercase "basic:1" and UUID-only USN (no service suffix).
func (l *Listener) buildDeviceResponse() string {
	fullUUID := uuidPrefix + l.info.Suffix
	return l.ssdpHeader() +
		"ST: urn:schemas-upnp-org:device:basic:1\r\n" +
		"USN: uuid:" + fullUUID + "\r\n\r\n"
}

// sendNotify broadcasts SSDP NOTIFY for three UPnP service types.
func (l *Listener) sendNotify() {
	ssdpUDPAddr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return
	}
	// Bind outbound socket to bridge IP so NOTIFY exits on the correct interface.
	// When IP is unknown (0.0.0.0) use nil so the kernel picks the route — better
	// than accidentally binding to an unspecified address.
	var localAddr *net.UDPAddr
	if l.info.IP != "" && l.info.IP != "0.0.0.0" {
		localAddr = &net.UDPAddr{IP: net.ParseIP(l.info.IP)}
	}
	conn, err := net.DialUDP("udp4", localAddr, ssdpUDPAddr)
	if err != nil {
		logbuf.Global.Debug("SSDP NOTIFY dial error: %v", err)
		return
	}
	defer conn.Close()
	fullUUID := uuidPrefix + l.info.Suffix
	notifies := []struct{ nt, usn string }{
		{"upnp:rootdevice", "uuid:" + fullUUID + "::upnp:rootdevice"},
		{"uuid:" + fullUUID, "uuid:" + fullUUID},
		{"urn:schemas-upnp-org:device:basic:1", "uuid:" + fullUUID},
	}
	for _, n := range notifies {
		msg := "NOTIFY * HTTP/1.1\r\n" +
			"HOST: 239.255.255.250:1900\r\n" +
			"CACHE-CONTROL: max-age=60\r\n" +
			"LOCATION: " + l.location() + "\r\n" +
			"NT: " + n.nt + "\r\n" +
			"NTS: ssdp:alive\r\n" +
			"SERVER: FreeRTOS/6.0.5, UPnP/1.0, IpBridge/" + apiVersion + "\r\n" +
			"USN: " + n.usn + "\r\n" +
			"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
			fmt.Sprintf("BOOTID.UPNP.ORG: %d\r\n", startupTime) +
			"CONFIGID.UPNP.ORG: 1\r\n" +
			"\r\n"
		for i := 0; i < 2; i++ {
			conn.Write([]byte(msg))
		}
		time.Sleep(50 * time.Millisecond)
	}
	logbuf.Global.Info("SSDP NOTIFY sent (LOCATION: %s)", l.location())
}

// RegisterDescription registers /description.xml and icon stubs on the HTTP mux.
func RegisterDescription(mux *http.ServeMux, info identity.BridgeInfo) {
	desc := buildDescriptionXML(info)
	mux.HandleFunc("/description.xml", func(w http.ResponseWriter, r *http.Request) {
		logbuf.Global.Debug("description.xml fetched from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, desc)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/hue_logo_0.png", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("/hue_logo_3.png", func(w http.ResponseWriter, r *http.Request) {})
}

// buildDescriptionXML returns the UPnP device description, matching HA's format:
// no iconList, no presentationURL — just the fields Alexa actually needs.
func buildDescriptionXML(info identity.BridgeInfo) string {
	fullUUID := uuidPrefix + info.Suffix
	urlBase := fmt.Sprintf("http://%s:%d/", info.IP, info.Port)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" ?>
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
</device>
</root>
`, urlBase, info.IP, modelID, info.BridgeID, fullUUID)
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimRight(s[:idx], "\r")
	}
	return strings.TrimRight(s, "\r\n")
}

func extractHeader(msg, key string) string {
	search := strings.ToLower(key) + ":"
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(strings.ToLower(line), search) {
			return strings.TrimSpace(line[len(search):])
		}
	}
	return ""
}
