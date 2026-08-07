package upnp

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/BattloXX/EchoLox/internal/discovery"
	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/logbuf"
)

const (
	ssdpAddr   = "239.255.255.250:1900"
	apiVersion = "1.47.0"
	modelID    = "BSB002"
	uuidPrefix = "2f402f80-da50-11e1-9b23-"

	// ntRoot / ntBasic are the three NT/ST types a real Hue bridge announces.
	// Newer Alexa firmware expects all three.
	ntRoot  = "upnp:rootdevice"
	ntBasic = "urn:schemas-upnp-org:device:Basic:1"
)

// Listener listens for SSDP M-SEARCH packets and responds to Alexa discovery.
type Listener struct {
	info          identity.BridgeInfo
	notifyCh      chan struct{} // send to trigger an immediate NOTIFY burst
	responseMu    sync.Mutex
	lastResponses map[string]time.Time
	responseSlots chan struct{}
}

const responseInterval = time.Second

func NewListener(info identity.BridgeInfo) *Listener {
	return &Listener{
		info:          info,
		notifyCh:      make(chan struct{}, 1),
		lastResponses: make(map[string]time.Time),
		responseSlots: make(chan struct{}, 16),
	}
}

// TriggerNotify requests an immediate SSDP NOTIFY burst (non-blocking).
func (l *Listener) TriggerNotify() {
	select {
	case l.notifyCh <- struct{}{}:
	default:
	}
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

func (l *Listener) fullUUID() string {
	return uuidPrefix + l.info.Suffix
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

	// Startup burst + periodic announce every 5 min.
	go l.notifyLoop()

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
			ua := extractHeader(msg, "user-agent")
			discovery.RecordAlexa(src.IP.String(), ua)
			st := extractHeader(msg, "st")
			if l.allowResponse(src.IP.String(), time.Now()) {
				logbuf.Global.Info("SSDP M-SEARCH from %s (ST: %s) — responding (LOCATION: %s)", src, st, l.location())
				select {
				case l.responseSlots <- struct{}{}:
					go func() {
						defer func() { <-l.responseSlots }()
						l.respond(src, st)
					}()
				default:
					logbuf.Global.Debug("SSDP response dropped: responder pool is full")
				}
			} else {
				logbuf.Global.Debug("SSDP response to %s rate-limited", src.IP)
			}
		}
	}
}

func (l *Listener) allowResponse(ip string, now time.Time) bool {
	l.responseMu.Lock()
	defer l.responseMu.Unlock()
	if last, ok := l.lastResponses[ip]; ok && now.Sub(last) < responseInterval {
		return false
	}
	l.lastResponses[ip] = now
	for key, last := range l.lastResponses {
		if now.Sub(last) > time.Minute {
			delete(l.lastResponses, key)
		}
	}
	return true
}

// notifyLoop sends an initial 3-packet burst then repeats every 5 min (or on demand).
func (l *Listener) notifyLoop() {
	time.Sleep(2 * time.Second)
	l.sendNotifyBurst()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.sendNotifyBurst()
		case <-l.notifyCh:
			l.sendNotifyBurst()
		}
	}
}

// sendNotifyBurst sends the three SSDP NOTIFY variants that a real Hue bridge sends.
func (l *Listener) sendNotifyBurst() {
	nts := []string{
		ntRoot,
		"uuid:" + l.fullUUID(),
		ntBasic,
	}
	for _, nt := range nts {
		l.sendNotify(nt)
		time.Sleep(time.Second)
	}
	logbuf.Global.Info("SSDP NOTIFY burst sent (3 variants, LOCATION: %s)", l.location())
}

// isMSearch returns true for any M-SEARCH with MAN: "ssdp:discover".
func (l *Listener) isMSearch(msg string) bool {
	if !strings.HasPrefix(msg, "M-SEARCH") {
		return false
	}
	return strings.Contains(strings.ToLower(msg), "ssdp:discover")
}

// respond sends a unicast 200 OK that mirrors the ST from the M-SEARCH request.
func (l *Listener) respond(src *net.UDPAddr, st string) {
	// Normalise: treat ssdp:all as Basic:1; fall back to Basic:1 for unknown types.
	switch strings.ToLower(strings.TrimSpace(st)) {
	case "ssdp:all", ntBasic, "":
		l.sendResponse(src, ntBasic)
	case ntRoot:
		l.sendResponse(src, ntRoot)
	default:
		l.sendResponse(src, ntBasic)
	}
}

func (l *Listener) sendResponse(src *net.UDPAddr, st string) {
	conn, err := net.DialUDP("udp4", nil, src)
	if err != nil {
		logbuf.Global.Debug("SSDP unicast dial: %v", err)
		return
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(l.buildResponse(st))); err != nil {
		logbuf.Global.Debug("SSDP unicast write: %v", err)
	} else {
		logbuf.Global.Debug("SSDP 200 OK sent to %s (ST: %s)", src, st)
	}
}

// buildResponse returns the SSDP 200 OK for the given ST.
func (l *Listener) buildResponse(st string) string {
	usn := "uuid:" + l.fullUUID() + "::" + st
	if st == "uuid:"+l.fullUUID() {
		usn = "uuid:" + l.fullUUID()
	}
	return "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=100\r\n" +
		"EXT:\r\n" +
		"LOCATION: " + l.location() + "\r\n" +
		"SERVER: Linux/3.14.0 UPnP/1.0 IpBridge/" + apiVersion + "\r\n" +
		"ST: " + st + "\r\n" +
		"USN: " + usn + "\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		"\r\n"
}

// sendNotify broadcasts a single SSDP NOTIFY for the given NT.
func (l *Listener) sendNotify(nt string) {
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

	usn := "uuid:" + l.fullUUID() + "::" + nt
	if nt == "uuid:"+l.fullUUID() {
		usn = "uuid:" + l.fullUUID()
	}
	notify := "NOTIFY * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"LOCATION: " + l.location() + "\r\n" +
		"NT: " + nt + "\r\n" +
		"NTS: ssdp:alive\r\n" +
		"SERVER: Linux/3.14.0 UPnP/1.0 IpBridge/" + apiVersion + "\r\n" +
		"USN: " + usn + "\r\n" +
		"hue-bridgeid: " + l.info.BridgeID + "\r\n" +
		"\r\n"
	if _, err := conn.Write([]byte(notify)); err != nil {
		logbuf.Global.Debug("SSDP NOTIFY write error (NT: %s): %v", nt, err)
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

func extractHeader(msg, key string) string {
	search := strings.ToLower(key) + ":"
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(strings.ToLower(line), search) {
			return strings.TrimSpace(line[len(search):])
		}
	}
	return ""
}
