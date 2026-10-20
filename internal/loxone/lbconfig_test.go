package loxone

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLBMiniserverCredentials(t *testing.T) {
	for _, tt := range []struct {
		name, config, user, pass string
	}{
		{"encoded", `{"Admin":"admin%40home","Pass":"p%21%2B%25%3A%2F%20%C3%A4"}`, "admin@home", "p!+%:/ ä"},
		{"plain", `{"Admin":"admin","Pass":"password"}`, "admin", "password"},
		{"literal plus", `{"Admin":"admin+home","Pass":"p+word"}`, "admin+home", "p+word"},
		{"decode once", `{"Admin":"admin","Pass":"p%2521"}`, "admin", "p%21"},
		{"raw preferred", `{"Admin":"old","Pass":"old","Admin_raw":"admin+home","Pass_raw":"p%21+%"}`, "admin+home", "p%21+%"},
		{"raw empty", `{"Admin":"old","Pass":"old","Admin_raw":"","Pass_raw":""}`, "", ""},
		{"mixed", `{"Admin":"admin%40home","Pass":"old","Pass_raw":"p%21"}`, "admin@home", "p%21"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ms LBMiniserver
			if err := json.Unmarshal([]byte(tt.config), &ms); err != nil {
				t.Fatal(err)
			}
			user, pass := NewClient(&ms, "http", 7777).Credentials()
			if user != tt.user || pass != tt.pass {
				t.Fatal("credentials differ from expected decoded values")
			}
		})
	}
}

func TestLBMiniserverInvalidCredentialEncoding(t *testing.T) {
	var ms LBMiniserver
	err := json.Unmarshal([]byte(`{"Admin":"admin","Pass":"secret%broken"}`), &ms)
	if err == nil || !strings.Contains(err.Error(), "Pass") {
		t.Fatalf("expected credential encoding error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "%b") {
		t.Fatal("error exposes credential contents")
	}
}

func TestLoxBerryCredentialsAuthenticateAllHTTPPaths(t *testing.T) {
	const user, pass = "admin@home", "secret!+%21"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok := r.BasicAuth()
		if !ok || gotUser != user || gotPass != pass {
			http.Error(w, "access denied", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/data/LoxAPP3.json" {
			w.Write([]byte(`{"controls":{"vi":{"name":"echolox_test_on","type":"VirtualInput"}}}`))
		} else {
			w.Write([]byte(`{"LL":{"Code":"200"}}`))
		}
	}))
	defer srv.Close()
	host, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	lbHome := t.TempDir()
	configDir := filepath.Join(lbHome, "config", "system")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(map[string]interface{}{
		"Miniserver": map[string]interface{}{
			"1": map[string]string{"Ipaddress": host, "Port": port, "Admin": "admin%40home", "Pass": "secret%21%2B%2521"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "general.json"), config, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LBHOMEDIR", lbHome)
	servers, err := ReadLoxBerryMiniservers()
	if err != nil {
		t.Fatal(err)
	}
	ms := servers["1"]
	c := NewClient(&ms, "http", 7777)
	v := NewVerifier(c)
	if err := v.RefreshCache(); err != nil {
		t.Fatalf("Miniserver test: %v", err)
	}
	if proposals, err := FetchVIProposals(c); err != nil || len(proposals) != 1 {
		t.Fatalf("discovery: got %d proposals, error %v", len(proposals), err)
	}
	if status := v.doProbe("echolox_test_on"); status != StatusOK {
		t.Fatalf("VI probe: %s", status)
	}
	if err := c.Send("echolox_test_on", "1"); err != nil {
		t.Fatalf("device test: %v", err)
	}
}
