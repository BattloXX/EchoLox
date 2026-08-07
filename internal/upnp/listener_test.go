package upnp

import (
	"testing"
	"time"

	"github.com/BattloXX/EchoLox/internal/identity"
)

func TestAllowResponseRateLimitsByIP(t *testing.T) {
	l := NewListener(identity.BridgeInfo{})
	now := time.Now()
	if !l.allowResponse("192.0.2.1", now) {
		t.Fatal("first response should be allowed")
	}
	if l.allowResponse("192.0.2.1", now.Add(responseInterval/2)) {
		t.Fatal("response inside rate-limit window should be rejected")
	}
	if !l.allowResponse("192.0.2.1", now.Add(responseInterval)) {
		t.Fatal("response after rate-limit window should be allowed")
	}
	if !l.allowResponse("192.0.2.2", now.Add(responseInterval/2)) {
		t.Fatal("a different source IP should be allowed")
	}
}
