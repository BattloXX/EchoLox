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
	IPAddress string `json:"IPAddress"`
	Port      string `json:"Port"`
	Admin     string `json:"Admin"`
	Pass      string `json:"Pass"`
}

func ReadLoxBerryMiniservers() (map[string]LBMiniserver, error) {
	path := findMiniserverJSON()
	log.Printf("Reading miniserver config from: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("miniserver.json not found at %s: %w", path, err)
	}

	// Try nested {"Miniserver": {"1": {...}}} format
	var nested struct {
		Miniserver map[string]LBMiniserver `json:"Miniserver"`
	}
	if err := json.Unmarshal(data, &nested); err == nil && len(nested.Miniserver) > 0 {
		log.Printf("Found %d miniserver(s) in LoxBerry config", len(nested.Miniserver))
		return nested.Miniserver, nil
	}

	// Try flat {"1": {...}} format
	var flat map[string]LBMiniserver
	if err := json.Unmarshal(data, &flat); err == nil && len(flat) > 0 {
		log.Printf("Found %d miniserver(s) in LoxBerry config (flat format)", len(flat))
		return flat, nil
	}

	return nil, fmt.Errorf("miniserver.json at %s has unexpected format", path)
}

func findMiniserverJSON() string {
	candidates := []string{}

	if v := os.Getenv("LBSCFG"); v != "" {
		candidates = append(candidates, filepath.Join(v, "miniserver.json"))
	}
	if v := os.Getenv("LBHOMEDIR"); v != "" {
		candidates = append(candidates, filepath.Join(v, "config", "system", "miniserver.json"))
	}
	candidates = append(candidates, "/opt/loxberry/config/system/miniserver.json")

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[len(candidates)-1]
}
