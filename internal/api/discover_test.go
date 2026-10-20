package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BattloXX/EchoLox/internal/loxone"
)

func TestDiscoverExplicitInputWithoutAppEntry(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jdev/sps/io/echolox_licht_kueche/state" {
			t.Errorf("unexpected request (must only read status): %s", r.URL.Path)
		}
		user, pass, _ := r.BasicAuth()
		if user != "test" || pass != "secret" {
			t.Error("missing credentials")
		}
		called = true
		w.Write([]byte(`{"LL":{"Code":200,"value":"0"}}`))
	}))
	defer srv.Close()
	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	h := &Handler{lox: loxone.NewClient(&loxone.LBMiniserver{
		IPAddress: host, Port: port, Admin: "test", Pass: "secret",
	}, "http", 7777)}
	response := httptest.NewRecorder()
	h.handleDiscoverLoxone(response, httptest.NewRequest(http.MethodGet,
		"/echolox/api/discover/loxone?name=echolox_licht_kueche", nil))
	var got map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !called || response.Code != 200 || got["status"] != "ok" || got["name"] != "echolox_licht_kueche" {
		t.Fatalf("unexpected probe response: %s", response.Body.String())
	}
}
