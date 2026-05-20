package loxone

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type VIStatus string

const (
	StatusOK           VIStatus = "ok"
	StatusNotFound     VIStatus = "not_found"
	StatusAccessDenied VIStatus = "access_denied"
	StatusNotSent      VIStatus = "not_sent"
)

type VIInfo struct {
	Name   string
	Status VIStatus
}

// loxControl mirrors one entry from LoxAPP3.json controls / subControls.
type loxControl struct {
	Name        string                `json:"name"`
	Type        string                `json:"type"`
	SubControls map[string]loxControl `json:"subControls"`
}

type Verifier struct {
	client    *Client
	http      *http.Client
	mu        sync.Mutex
	cachedVIs map[string]bool     // name → exists (from LoxAPP3.json, recursive)
	probeCache map[string]probeResult // per-VI probe results (short TTL)
	lastFetch time.Time
}

type probeResult struct {
	status VIStatus
	at     time.Time
}

const probeTTL = 30 * time.Second

func NewVerifier(c *Client) *Verifier {
	return &Verifier{
		client: c,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (v *Verifier) RefreshCache() error {
	if v.client == nil {
		return fmt.Errorf("no loxone client configured")
	}
	url := v.client.BaseURL() + "/data/LoxAPP3.json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	user, pass := v.client.Credentials()
	req.SetBasicAuth(user, pass)
	resp, err := v.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("access denied")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	// Detect Loxone LL error wrapper (HTTP 200 with {"LL":{"Code":"401",...}}).
	if llData, ok := raw["LL"]; ok {
		var ll struct {
			Code string `json:"Code"`
		}
		if json.Unmarshal(llData, &ll) == nil && ll.Code != "" && ll.Code != "200" {
			return fmt.Errorf("access denied")
		}
	}
	var controlsJSON json.RawMessage
	for k, val := range raw {
		if strings.EqualFold(k, "controls") {
			controlsJSON = val
			break
		}
	}
	if controlsJSON == nil {
		return fmt.Errorf("LoxAPP3.json: no controls field")
	}
	var controls map[string]loxControl
	if err := json.Unmarshal(controlsJSON, &controls); err != nil {
		return err
	}
	cache := make(map[string]bool)
	for _, ctrl := range controls {
		indexControl(ctrl, cache)
	}
	v.mu.Lock()
	v.cachedVIs = cache
	v.probeCache = make(map[string]probeResult)
	v.lastFetch = time.Now()
	v.mu.Unlock()
	return nil
}

// indexControl recursively indexes the name of ctrl and all its subControls.
func indexControl(ctrl loxControl, out map[string]bool) {
	if ctrl.Name != "" {
		out[ctrl.Name] = true
	}
	for _, sub := range ctrl.SubControls {
		indexControl(sub, out)
	}
}

func (v *Verifier) CheckVI(name string) VIStatus {
	v.mu.Lock()
	if v.cachedVIs == nil {
		v.mu.Unlock()
		if err := v.RefreshCache(); err != nil {
			if err.Error() == "access denied" {
				return StatusAccessDenied
			}
			return StatusNotFound
		}
		v.mu.Lock()
	}
	found := v.cachedVIs[name]
	v.mu.Unlock()

	if found {
		return StatusOK
	}
	// Cache says not found — fall back to a per-VI probe so the status page is
	// correct even when the Loxone firmware returns VIs at a different JSON path.
	return v.probeVI(name)
}

// probeVI queries /jdev/sps/io/<name> directly to confirm VI existence.
// Results are cached for probeTTL to avoid hammering the Miniserver when the
// Status page iterates all VIs.
func (v *Verifier) probeVI(name string) VIStatus {
	if v.client == nil {
		return StatusNotFound
	}
	v.mu.Lock()
	if v.probeCache == nil {
		v.probeCache = make(map[string]probeResult)
	}
	if r, ok := v.probeCache[name]; ok && time.Since(r.at) < probeTTL {
		v.mu.Unlock()
		return r.status
	}
	v.mu.Unlock()

	status := v.doProbe(name)

	v.mu.Lock()
	v.probeCache[name] = probeResult{status: status, at: time.Now()}
	v.mu.Unlock()
	return status
}

func (v *Verifier) doProbe(name string) VIStatus {
	probeURL := v.client.BaseURL() + "/jdev/sps/io/" + url.PathEscape(name)
	req, err := http.NewRequest("GET", probeURL, nil)
	if err != nil {
		return StatusNotFound
	}
	user, pass := v.client.Credentials()
	req.SetBasicAuth(user, pass)
	resp, err := v.http.Do(req)
	if err != nil {
		return StatusNotFound
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return StatusAccessDenied
	}
	// Loxone returns HTTP 200 for both found and not-found cases;
	// the result code lives inside the JSON body.
	var ll struct {
		LL struct {
			Code string `json:"Code"`
		} `json:"LL"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ll); err != nil {
		return StatusNotFound
	}
	if ll.LL.Code == "200" {
		return StatusOK
	}
	return StatusNotFound
}
