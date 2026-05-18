package loxone

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildLoxAPP3 returns a minimal LoxAPP3.json payload with the given controls.
func buildLoxAPP3(controls map[string]loxControl) []byte {
	body, _ := json.Marshal(map[string]interface{}{
		"controls": controls,
	})
	return body
}

// testVerifier creates a Verifier backed by a test HTTP server.
func testVerifier(t *testing.T, loxAPP3Body []byte, jdevHandler http.HandlerFunc) (*Verifier, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/data/LoxAPP3.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(loxAPP3Body)
	})
	if jdevHandler != nil {
		mux.HandleFunc("/jdev/sps/io/", jdevHandler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ms := &LBMiniserver{IPAddress: host, Port: port, Admin: "user", Pass: "pass"}
	client := NewClient(ms, "http", 7777)
	v := NewVerifier(client)
	return v, srv
}

func TestCheckVI_TopLevelControl(t *testing.T) {
	body := buildLoxAPP3(map[string]loxControl{
		"uuid-1": {Name: "echolox_wohnzimmer_licht_on", Type: "VirtualInput"},
	})
	v, _ := testVerifier(t, body, nil)

	if s := v.CheckVI("echolox_wohnzimmer_licht_on"); s != StatusOK {
		t.Errorf("expected ok, got %s", s)
	}
	if s := v.CheckVI("echolox_nonexistent"); s == StatusOK {
		t.Errorf("expected not ok for missing VI, got %s", s)
	}
}

func TestCheckVI_NestedSubControl(t *testing.T) {
	body := buildLoxAPP3(map[string]loxControl{
		"uuid-parent": {
			Name: "Wohnzimmer",
			Type: "LightControllerV2",
			SubControls: map[string]loxControl{
				"uuid-sub-1": {Name: "echolox_wohnzimmer_licht_on", Type: "Switch"},
				"uuid-sub-2": {Name: "echolox_wohnzimmer_licht_off", Type: "Switch"},
			},
		},
	})
	// No probe handler — VI must be found via subControls parse
	v, _ := testVerifier(t, body, nil)

	for _, vi := range []string{"echolox_wohnzimmer_licht_on", "echolox_wohnzimmer_licht_off"} {
		if s := v.CheckVI(vi); s != StatusOK {
			t.Errorf("expected ok for nested VI %q, got %s", vi, s)
		}
	}
}

func TestCheckVI_ProbeFallback(t *testing.T) {
	// Empty LoxAPP3 — VI not in cache; must fall back to jdev probe
	body := buildLoxAPP3(map[string]loxControl{})
	probed := map[string]bool{}
	jdevHandler := func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/jdev/sps/io/"):]
		probed[name] = true
		code := "404"
		if name == "echolox_exists" {
			code = "200"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"LL": map[string]string{"Code": code}})
	}
	v, _ := testVerifier(t, body, jdevHandler)

	if s := v.CheckVI("echolox_exists"); s != StatusOK {
		t.Errorf("probe fallback: expected ok, got %s", s)
	}
	if s := v.CheckVI("echolox_missing"); s != StatusNotFound {
		t.Errorf("probe fallback: expected not_found, got %s", s)
	}
	if !probed["echolox_exists"] {
		t.Error("expected jdev probe to be called for echolox_exists")
	}
}

func TestCheckVI_ProbeResultCached(t *testing.T) {
	body := buildLoxAPP3(map[string]loxControl{})
	callCount := 0
	jdevHandler := func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]interface{}{"LL": map[string]string{"Code": "200"}})
	}
	v, _ := testVerifier(t, body, jdevHandler)

	v.CheckVI("echolox_test")
	v.CheckVI("echolox_test")
	if callCount != 1 {
		t.Errorf("expected probe to be called once (cached), got %d", callCount)
	}
}
