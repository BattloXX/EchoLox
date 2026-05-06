// Package identity holds the stable Hue bridge identity (UUID, bridgeID, MAC).
// Defined here so both bridge and hue packages can import it without cycles.
package identity

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// BridgeInfo is the stable Hue identity derived deterministically from the bridge IP.
type BridgeInfo struct {
	IP       string // IP address Alexa reaches
	Port     int
	UUID     string // full UUID: 2f402f80-da50-11e1-9b23-{12hex}
	BridgeID string // 001788FFFE{6hex}, 16 chars uppercase
	MAC      string // 00:17:88:xx:xx:xx
	Suffix   string // 12-char lowercase hex suffix (last part of UUID)
}

// New derives a stable bridge identity from the IP address.
// Using Philips OUI (001788) so Alexa recognises it as a genuine Hue bridge.
func New(ip string, port int) BridgeInfo {
	h := md5.Sum([]byte("echolox-bridge:" + ip))
	suffix := fmt.Sprintf("%012x", h[:6])
	bridgeID := strings.ToUpper("001788fffe" + suffix[6:])
	mac := fmt.Sprintf("00:17:88:%02x:%02x:%02x", h[3], h[4], h[5])
	return BridgeInfo{
		IP:       ip,
		Port:     port,
		UUID:     "2f402f80-da50-11e1-9b23-" + suffix,
		BridgeID: bridgeID,
		MAC:      mac,
		Suffix:   suffix,
	}
}
