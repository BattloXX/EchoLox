package loxone

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"
)

// VIProposal represents a group of echolox_* Virtual Inputs that belong to one device.
type VIProposal struct {
	Base        string   `json:"base"`         // e.g. "echolox_wohnzimmer_licht"
	DisplayName string   `json:"display_name"` // e.g. "Wohnzimmer Licht"
	Type        string   `json:"type"`         // "switch" / "dimmer" / "color" / "scene"
	SwitchMode  string   `json:"switch_mode"`  // "onoff" / "impulse"
	VIs         []string `json:"vis"`          // all VI names in this group
}

// FetchVIProposals queries the Miniserver for all echolox_* Virtual Inputs and
// groups them into device proposals by analysing the known suffix patterns.
func FetchVIProposals(c *Client) ([]VIProposal, error) {
	url := c.BaseURL() + "/data/LoxAPP3.json"
	log.Printf("viprobe: fetching %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	user, pass := c.Credentials()
	req.SetBasicAuth(user, pass)
	log.Printf("viprobe: using credentials user=%q", user)

	hc := &http.Client{Timeout: 10 * time.Second}
	resp, err := hc.Do(req)
	if err != nil {
		log.Printf("viprobe: HTTP request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("viprobe: HTTP status %d", resp.StatusCode)

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("Zugriff verweigert (401) — Benutzername/Passwort prüfen")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Miniserver antwortete mit HTTP %d", resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		log.Printf("viprobe: JSON decode error: %v", err)
		return nil, fmt.Errorf("LoxAPP3.json parsen: %w", err)
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	log.Printf("viprobe: top-level JSON keys: %v", keys)

	// Detect Loxone LL error wrapper — firmware returns HTTP 200 with
	// {"LL":{"Code":"401","value":"..."}} instead of a real HTTP 401.
	if llData, ok := raw["LL"]; ok {
		var ll struct {
			Code  string `json:"Code"`
			Value string `json:"value"`
		}
		if json.Unmarshal(llData, &ll) == nil {
			log.Printf("viprobe: LL wrapper detected: Code=%q Value=%q", ll.Code, ll.Value)
			if ll.Code != "" && ll.Code != "200" {
				return nil, fmt.Errorf("Miniserver-Fehler (Code %s) — Zugangsdaten prüfen: %s", ll.Code, ll.Value)
			}
		}
	}

	// Find the controls key case-insensitively ("Controls" in real firmware).
	var controlsJSON json.RawMessage
	var controlsKey string
	for k, v := range raw {
		if strings.EqualFold(k, "controls") {
			controlsJSON = v
			controlsKey = k
			break
		}
	}
	if controlsJSON == nil {
		log.Printf("viprobe: no controls key found in keys: %v", keys)
		return nil, fmt.Errorf("LoxAPP3.json enthält kein 'Controls'-Feld (vorhandene Schlüssel: %v)", keys)
	}
	log.Printf("viprobe: found controls under key %q", controlsKey)

	var controls map[string]loxControl
	if err := json.Unmarshal(controlsJSON, &controls); err != nil {
		log.Printf("viprobe: controls unmarshal error: %v", err)
		return nil, fmt.Errorf("LoxAPP3.json Controls parsen: %w", err)
	}
	log.Printf("viprobe: parsed %d top-level controls", len(controls))

	var echoloxVIs []string
	for _, ctrl := range controls {
		collectEcholoxVIs(ctrl, &echoloxVIs)
	}
	sort.Strings(echoloxVIs)
	log.Printf("viprobe: found %d echolox_* VIs: %v", len(echoloxVIs), echoloxVIs)

	proposals := groupEcholoxVIs(echoloxVIs)
	log.Printf("viprobe: grouped into %d proposals", len(proposals))
	return proposals, nil
}

// collectEcholoxVIs appends all echolox_* names found in ctrl and its subControls.
// The prefix check is case-insensitive so that VIs named "EchoLox_*" or
// "ECHOLOX_*" are found as well; the original name is preserved for HTTP commands.
func collectEcholoxVIs(ctrl loxControl, out *[]string) {
	if strings.HasPrefix(strings.ToLower(ctrl.Name), "echolox_") {
		*out = append(*out, ctrl.Name)
	}
	for _, sub := range ctrl.SubControls {
		collectEcholoxVIs(sub, out)
	}
}

// knownSuffixes are unambiguous — no need to check for a complementary VI.
var knownSuffixes = []string{"_activate", "_brightness", "_hue", "_saturation"}

// viBase returns the group base for a given VI name.
// _on/_off are only treated as suffixes if the complementary VI also exists,
// to avoid misidentifying a device named "foo on" as an Impulse-mode "foo".
func viBase(vi string, viSet map[string]bool) string {
	for _, suf := range knownSuffixes {
		if strings.HasSuffix(vi, suf) {
			return vi[:len(vi)-len(suf)]
		}
	}
	if strings.HasSuffix(vi, "_on") {
		candidate := vi[:len(vi)-3]
		if viSet[candidate+"_off"] {
			return candidate
		}
	}
	if strings.HasSuffix(vi, "_off") {
		candidate := vi[:len(vi)-4]
		if viSet[candidate+"_on"] {
			return candidate
		}
	}
	// No suffix matched — the VI itself is the on-VI in OnOff mode.
	return vi
}

func groupEcholoxVIs(vis []string) []VIProposal {
	viSet := make(map[string]bool, len(vis))
	for _, v := range vis {
		viSet[v] = true
	}

	byBase := make(map[string][]string)
	var basesOrdered []string
	seenBase := make(map[string]bool)

	for _, vi := range vis {
		base := viBase(vi, viSet)
		if !seenBase[base] {
			seenBase[base] = true
			basesOrdered = append(basesOrdered, base)
		}
		byBase[base] = append(byBase[base], vi)
	}

	proposals := make([]VIProposal, 0, len(basesOrdered))
	for _, base := range basesOrdered {
		group := byBase[base]
		gSet := make(map[string]bool, len(group))
		for _, v := range group {
			gSet[v] = true
		}

		dtype := "switch"
		mode := "onoff"

		switch {
		case gSet[base+"_activate"]:
			dtype = "scene"
		case gSet[base+"_hue"] || gSet[base+"_saturation"]:
			dtype = "color"
		case gSet[base+"_brightness"]:
			dtype = "dimmer"
		}

		if gSet[base+"_on"] && gSet[base+"_off"] {
			mode = "impulse"
		}

		sort.Strings(group)
		proposals = append(proposals, VIProposal{
			Base:        base,
			DisplayName: viBaseToDisplayName(base),
			Type:        dtype,
			SwitchMode:  mode,
			VIs:         group,
		})
	}
	return proposals
}

func viBaseToDisplayName(base string) string {
	// Strip the echolox_ prefix case-insensitively so "EchoLox_Licht" and
	// "echolox_licht" both produce the same display name.
	name := base
	if len(base) >= 8 && strings.ToLower(base[:8]) == "echolox_" {
		name = base[8:]
	}
	words := strings.Split(name, "_")
	for i, w := range words {
		if len(w) > 0 {
			r := []rune(w)
			r[0] = unicode.ToUpper(r[0])
			words[i] = string(r)
		}
	}
	return strings.Join(words, " ")
}
