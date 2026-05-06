package loxone

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type LBMiniserver struct {
	Name      string `json:"Name"`
	IPAddress string `json:"Ipaddress"` // LoxBerry general.json uses lowercase 'p'
	Port      string `json:"Port"`
	Admin     string `json:"Admin"`
	Pass      string `json:"Pass"`
}

func ReadLoxBerryMiniservers() (map[string]LBMiniserver, error) {
	path := findGeneralJSON()
	log.Printf("Reading LoxBerry config from: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("general.json not found at %s: %w", path, err)
	}

	// LoxBerry general.json: {"Miniserver": {"1": {"Ipaddress": "...", ...}}}
	var nested struct {
		Miniserver map[string]LBMiniserver `json:"Miniserver"`
	}
	if err := json.Unmarshal(data, &nested); err == nil && len(nested.Miniserver) > 0 {
		log.Printf("Found %d miniserver(s) in LoxBerry config", len(nested.Miniserver))
		return nested.Miniserver, nil
	}

	return nil, fmt.Errorf("general.json at %s: no Miniserver entries found", path)
}

func findGeneralJSON() string {
	if v := os.Getenv("LBHOMEDIR"); v != "" {
		p := filepath.Join(v, "config", "system", "general.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/opt/loxberry/config/system/general.json"
}
