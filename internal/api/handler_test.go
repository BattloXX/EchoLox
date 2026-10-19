package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BattloXX/EchoLox/internal/loxone"
)

func TestVerifyReturnsJSON(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantError  string
	}{
		{"unauthorized", 401, "access denied", 500, "access denied"},
		{"LL unauthorized", 200, `{"LL":{"Code":"401"}}`, 500, "access denied"},
		{"invalid response", 200, "not JSON", 500, ""},
		{"success", 200, `{"controls":{}}`, 200, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			client := loxone.NewClient(&loxone.LBMiniserver{IPAddress: host, Port: port}, "http", 7777)
			h := &Handler{verifier: loxone.NewVerifier(client)}
			mux := http.NewServeMux()
			h.Register(mux)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/echolox/api/verify", nil))
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var result map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}
			if tt.wantStatus == http.StatusOK {
				if result["status"] != "ok" {
					t.Fatalf("expected success, got %v", result)
				}
			} else if result["error"] == "" || (tt.wantError != "" && result["error"] != tt.wantError) {
				t.Fatalf("unexpected error response: %v", result)
			}
		})
	}
}
