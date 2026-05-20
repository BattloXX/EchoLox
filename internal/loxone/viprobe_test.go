package loxone

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testClient creates a Client backed by a test HTTP server serving LoxAPP3.json.
func testClient(t *testing.T, loxAPP3Body []byte) (*Client, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/data/LoxAPP3.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(loxAPP3Body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ms := &LBMiniserver{IPAddress: host, Port: port, Admin: "user", Pass: "pass"}
	return NewClient(ms, "http", 7777), srv
}

func TestFetchVIProposals_LowercasePrefix(t *testing.T) {
	body := buildLoxAPP3(map[string]loxControl{
		"u1": {Name: "echolox_licht_on", Type: "VirtualInput"},
		"u2": {Name: "echolox_licht_off", Type: "VirtualInput"},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(props))
	}
	if props[0].Type != "switch" || props[0].SwitchMode != "impulse" {
		t.Errorf("unexpected proposal: %+v", props[0])
	}
}

func TestFetchVIProposals_PlainNames(t *testing.T) {
	// VIs without any echolox_ prefix — should be found by type.
	body := buildLoxAPP3(map[string]loxControl{
		"u1": {Name: "Licht_on", Type: "VirtualInput"},
		"u2": {Name: "Licht_off", Type: "VirtualInput"},
		"u3": {Name: "Jalousie", Type: "VirtualInput"},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 2 {
		t.Fatalf("expected 2 proposals (Licht pair + Jalousie), got %d: %+v", len(props), props)
	}
}

func TestFetchVIProposals_NonVITypeIgnored(t *testing.T) {
	// Controls that are not VirtualInput type must NOT be included.
	body := buildLoxAPP3(map[string]loxControl{
		"u1": {Name: "Licht_on", Type: "Switch"},
		"u2": {Name: "Licht_off", Type: "Jalousie"},
		"u3": {Name: "Real_VI", Type: "VirtualInput"},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal (only VirtualInput), got %d: %+v", len(props), props)
	}
	if props[0].Base != "Real_VI" {
		t.Errorf("expected base Real_VI, got %q", props[0].Base)
	}
}

func TestFetchVIProposals_MixedCasePrefix(t *testing.T) {
	// VIs named with capital E — must still be found and grouped.
	body := buildLoxAPP3(map[string]loxControl{
		"u1": {Name: "EchoLox_Dimmer_on", Type: "VirtualInput"},
		"u2": {Name: "EchoLox_Dimmer_off", Type: "VirtualInput"},
		"u3": {Name: "EchoLox_Dimmer_brightness", Type: "VirtualInput"},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal for mixed-case prefix, got %d", len(props))
	}
	p := props[0]
	if p.Type != "dimmer" {
		t.Errorf("expected type dimmer, got %q", p.Type)
	}
	if p.SwitchMode != "impulse" {
		t.Errorf("expected impulse mode, got %q", p.SwitchMode)
	}
	if len(p.VIs) != 3 {
		t.Errorf("expected 3 VIs, got %d: %v", len(p.VIs), p.VIs)
	}
}

func TestFetchVIProposals_NestedSubControl(t *testing.T) {
	// VIs as subcontrols of a parent block (common Loxone structure).
	body := buildLoxAPP3(map[string]loxControl{
		"parent": {
			Name: "EchoLox Eingänge",
			Type: "VirtualInput",
			SubControls: map[string]loxControl{
				"c1": {Name: "echolox_steckdose_on", Type: "VirtualInputText"},
				"c2": {Name: "echolox_steckdose_off", Type: "VirtualInputText"},
			},
		},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal from nested subcontrols, got %d", len(props))
	}
}

func TestFetchVIProposals_LLErrorWrapper(t *testing.T) {
	// Loxone returns HTTP 200 with {"LL":{"Code":"401","value":"..."}} on auth failure.
	mux := http.NewServeMux()
	mux.HandleFunc("/data/LoxAPP3.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"LL":{"Code":"401","value":"unauthorized"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ms := &LBMiniserver{IPAddress: host, Port: port, Admin: "user", Pass: "pass"}
	c := NewClient(ms, "http", 7777)
	_, err := FetchVIProposals(c)
	if err == nil {
		t.Fatal("expected error for LL error wrapper, got nil")
	}
}

func TestFetchVIProposals_PascalCaseControls(t *testing.T) {
	// Real Loxone firmware uses "Controls" (PascalCase), not "controls".
	body, _ := json.Marshal(map[string]interface{}{
		"Controls": map[string]loxControl{
			"u1": {Name: "echolox_jalousie_on", Type: "VirtualInput"},
			"u2": {Name: "echolox_jalousie_off", Type: "VirtualInput"},
		},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal for PascalCase Controls key, got %d", len(props))
	}
}

func TestFetchVIProposals_DisplayName(t *testing.T) {
	body := buildLoxAPP3(map[string]loxControl{
		"u1": {Name: "EchoLox_Wohnzimmer_Licht", Type: "VirtualInput"},
	})
	c, _ := testClient(t, body)
	props, err := FetchVIProposals(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(props))
	}
	want := "Wohnzimmer Licht"
	if props[0].DisplayName != want {
		t.Errorf("display name: got %q, want %q", props[0].DisplayName, want)
	}
}
