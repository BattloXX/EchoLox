package upnp

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/logbuf"
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

func (l *Listener) discoveryPort() int {
	if l.info.DiscoveryPort > 0 {
		return l.info.DiscoveryPort
	}
	return l.info.Port
}

func (l *Listener) location() string {
	return fmt.Sprintf("http://%s:%d/description.xml", l.info.IP, l.discoveryPort())
}

func (l *Listener) Listen() {
	conn, err := listenMulticast()
	if err != nil {
		logbuf.Global.Info("SSDP listen error: %v — Alexa discovery disabled", err)
		return
	}
	defer conn.Close()
	conn.SetReadBuffer(65536)
	logbuf.Global.Info("SSDP listener on %s  bridgeid=%s  LOCATION=%s", ssdpAddr, l.info.BridgeID, l.location())

	// Send NOTIFY announcements on start and every 30 min
	go func() {
		time.Sleep(2 * time.Second)
		l.sendNotify()
		ticker := time.NewTicker(30 * time.Minute)
		for range ticker.C {
			l.sendNotify()
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
		if l.isMSearch(msg) {
			logbuf.Global.Info("SSDP M-SEARCH from %s — responding (LOCATION: %s)", src, l.location())
			go l.respond(src)
		}
	}
}

// isMSearch returns true for any M-SEARCH with MAN: "ssdp:discover".
// Responds to all such requests (Alexa uses Basic:1, but we accept ssdp:all too).
func (l *Listener) isMSearch(msg string) bool {
	if !strings.HasPrefix(msg, "M-SEARCH") {
		return false
	}
	return strings.Contains(strings.ToLower(msg), "ssdp:discover")
}

// respond sends a unicast 200 OK via a dedicated ephemeral socket.
func (l *Listener) respond(src *net.UDPAddr) {
	conn, err := net.DialUDP("udp4", nil, src)
	if err != nil {
		logbuf.Global.Debug("SSDP unicast dial: %v", err)
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(l.buildResponse())); err != nil {
		logbuf.Global.Debug("SSDP unicast write: %v", err)
	} else {
		logbuf.Global.Debug("SSDP 200 OK sent to %s", src)
	}
}

// buildResponse returns the SSDP 200 OK Alexa expects.
func (l *Listener) buildResponse() string {
	fullUUID := uuidPrefix + l.info.Suffix
	return "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=100\r\n" +
		"EXT:\r\n" +
		"LOCATION: " + l.location() + "\r\n" +
		"SERVER: Linux/3.14.0 UPnP/1.0 IpBridge/" + apiVersion + "\r\n" +
		"ST: urn:schemas-upnp-org:device:Basic:1\r\n" +
		"USN: uuid:" + fullUUID + "::urn:schemas-upnp-org:device:Basic:1\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		"\r\n"
}

// sendNotify broadcasts an SSDP NOTIFY so Alexa can find the bridge without M-SEARCH.
func (l *Listener) sendNotify() {
	ssdpUDPAddr, err := net.ResolveUDPAddr("udp4", ssdpAddr)
	if err != nil {
		return
	}
	conn, err := net.DialUDP("udp4", nil, ssdpUDPAddr)
	if err != nil {
		logbuf.Global.Debug("SSDP NOTIFY dial error: %v", err)
		return
	}
	defer conn.Close()
	fullUUID := uuidPrefix + l.info.Suffix
	notify := "NOTIFY * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"LOCATION: " + l.location() + "\r\n" +
		"NT: urn:schemas-upnp-org:device:Basic:1\r\n" +
		"NTS: ssdp:alive\r\n" +
		"SERVER: Linux/3.14.0 UPnP/1.0 IpBridge/" + apiVersion + "\r\n" +
		"USN: uuid:" + fullUUID + "::urn:schemas-upnp-org:device:Basic:1\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		"\r\n"
	if _, err := conn.Write([]byte(notify)); err != nil {
		logbuf.Global.Debug("SSDP NOTIFY write error: %v", err)
	} else {
		logbuf.Global.Info("SSDP NOTIFY sent (LOCATION: %s)", l.location())
	}
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

// buildDescriptionXML returns the UPnP description Alexa fetches after SSDP discovery.
// URLBase uses discovery_port so Alexa routes subsequent API calls correctly.
func buildDescriptionXML(info identity.BridgeInfo) string {
	fullUUID := uuidPrefix + info.Suffix
	dPort := info.DiscoveryPort
	if dPort <= 0 {
		dPort = info.Port
	}
	urlBase := fmt.Sprintf("http://%s:%d/", info.IP, dPort)
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

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimRight(s[:idx], "\r")
	}
	return strings.TrimRight(s, "\r\n")
}
