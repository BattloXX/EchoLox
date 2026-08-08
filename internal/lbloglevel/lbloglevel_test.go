package lbloglevel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReaderAndForwarding(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data", "system", "plugindatabase.json")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		loglevel  string
		wantLevel int
		wantInfo  bool
		wantDebug bool
	}{
		{name: "off", loglevel: `0`, wantLevel: 0},
		{name: "error", loglevel: `3`, wantLevel: 3, wantInfo: true},
		{name: "info", loglevel: `6`, wantLevel: 6, wantInfo: true},
		{name: "debug", loglevel: `7`, wantLevel: 7, wantInfo: true, wantDebug: true},
		{name: "disabled", loglevel: `-1`, wantLevel: -1},
		{name: "missing", wantLevel: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levelField := ""
			if tt.loglevel != "" {
				levelField = `, "loglevel": ` + tt.loglevel
			}
			fixture := `{"plugins":{"wrong":{"folder":"Other","loglevel":7},` +
				`"opaque-key":{"folder":"EchoLox"` + levelField + `}}}`
			if err := os.WriteFile(dbPath, []byte(fixture), 0644); err != nil {
				t.Fatal(err)
			}

			r := newReader(dbPath)
			r.refresh()
			if got := r.currentLevel(); got != tt.wantLevel {
				t.Fatalf("currentLevel() = %d, want %d", got, tt.wantLevel)
			}
			if got := shouldForward(r.currentLevel(), false); got != tt.wantInfo {
				t.Errorf("Info forwarding = %v, want %v", got, tt.wantInfo)
			}
			if got := shouldForward(r.currentLevel(), true); got != tt.wantDebug {
				t.Errorf("Debug forwarding = %v, want %v", got, tt.wantDebug)
			}
		})
	}
}
