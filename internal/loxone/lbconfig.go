package loxone

import (
	"encoding/json"
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
	lbscfg := os.Getenv("LBSCFG")
	if lbscfg == "" {
		lbscfg = "/opt/loxberry/config/system"
	}
	path := filepath.Join(lbscfg, "miniserver.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result struct {
		Miniserver map[string]LBMiniserver `json:"Miniserver"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.Miniserver, nil
}
