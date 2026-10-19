package loxone

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
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

// UnmarshalJSON normalizes LoxBerry's URI-encoded credentials for Basic Auth.
// Raw fields, when present, must be used verbatim (including literal '%' or '+').
func (ms *LBMiniserver) UnmarshalJSON(data []byte) error {
	type miniserver LBMiniserver
	var stored struct {
		miniserver
		AdminRaw *string `json:"Admin_raw"`
		PassRaw  *string `json:"Pass_raw"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	for _, field := range []struct {
		name  string
		value *string
		raw   *string
	}{
		{"Admin", &stored.Admin, stored.AdminRaw},
		{"Pass", &stored.Pass, stored.PassRaw},
	} {
		if field.raw != nil {
			*field.value = *field.raw
			continue
		}
		// LoxBerry uses URI escaping, not form encoding: '+' stays '+'.
		decoded, err := url.PathUnescape(*field.value)
		if err != nil {
			// Do not include the credential in errors or logs.
			return fmt.Errorf("Miniserver %s: invalid URI encoding", field.name)
		}
		*field.value = decoded
	}
	*ms = LBMiniserver(stored.miniserver)
	return nil
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
	if err := json.Unmarshal(data, &nested); err != nil {
		return nil, fmt.Errorf("cannot parse general.json at %s: %w", path, err)
	}
	if len(nested.Miniserver) > 0 {
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
