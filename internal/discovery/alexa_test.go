package discovery

import (
	"testing"
	"time"
)

func TestRecordAlexaPrunesExpiredDevices(t *testing.T) {
	mu.Lock()
	devices = map[string]*AlexaDevice{
		"192.0.2.1": {IP: "192.0.2.1", LastSeen: time.Now().Add(-alexaDeviceTTL - time.Minute)},
	}
	mu.Unlock()

	RecordAlexa("192.0.2.2", "test-agent")
	got := AllAlexas()
	if len(got) != 1 || got[0].IP != "192.0.2.2" {
		t.Fatalf("expected only current device, got %#v", got)
	}

	mu.Lock()
	devices = map[string]*AlexaDevice{}
	mu.Unlock()
}
