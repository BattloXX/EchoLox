package loxone

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRejectsUnsupportedTransport(t *testing.T) {
	c := NewClient(&LBMiniserver{}, "mqtt", 7777)
	err := c.Send("test", "1")
	if err == nil || !strings.Contains(err.Error(), "unsupported Loxone transport") {
		t.Fatalf("expected unsupported transport error, got %v", err)
	}
}

func TestBaseURLJoinsIPv6HostAndPort(t *testing.T) {
	c := NewClient(&LBMiniserver{IPAddress: "::1", Port: "1234"}, "http", 7777)
	if got, want := c.BaseURL(), "http://[::1]:1234"; got != want {
		t.Fatalf("BaseURL() = %q, want %q", got, want)
	}
}

func TestBaseURLJoinsIPv4HostAndPort(t *testing.T) {
	c := NewClient(&LBMiniserver{IPAddress: "192.0.2.10", Port: "8080"}, "http", 7777)
	if got, want := c.BaseURL(), "http://192.0.2.10:8080"; got != want {
		t.Fatalf("BaseURL() = %q, want %q", got, want)
	}
}

func TestSendChecksMiniserverResult(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"XML success", `<LL Code="200" value="1"/>`, 200, false},
		{"JSON success", `{"LL":{"Code":"200"}}`, 200, false},
		{"numeric code", `{"LL":{"Code":200}}`, 200, false},
		{"XML missing input", `<LL Code="404"/>`, 200, true},
		{"JSON denied", `{"LL":{"Code":"403"}}`, 200, true},
		{"HTTP denied", "denied", 401, true},
		{"invalid response", "not a command response", 200, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/dev/sps/io/echolox_licht_kueche/pulse" {
					t.Errorf("unexpected command path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
			c := NewClient(&LBMiniserver{IPAddress: host, Port: port}, "http", 7777)
			err := c.Send("echolox_licht_kueche", "Impuls")
			if (err != nil) != tt.wantError {
				t.Fatalf("Send error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
