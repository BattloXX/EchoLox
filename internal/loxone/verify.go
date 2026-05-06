package loxone

import (
	"encoding/json"
	"fmt"
	"net/http"
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

type Verifier struct {
	client    *Client
	http      *http.Client
	cachedVIs map[string]bool // name → exists
	lastFetch time.Time
}

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
	var app struct {
		MSInfo struct {
			SerialNr string `json:"serialNr"`
		} `json:"msInfo"`
		Controls map[string]struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"controls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return err
	}
	v.cachedVIs = make(map[string]bool)
	for _, ctrl := range app.Controls {
		v.cachedVIs[ctrl.Name] = true
	}
	v.lastFetch = time.Now()
	return nil
}

func (v *Verifier) CheckVI(name string) VIStatus {
	if v.cachedVIs == nil {
		if err := v.RefreshCache(); err != nil {
			if err.Error() == "access denied" {
				return StatusAccessDenied
			}
			return StatusNotFound
		}
	}
	if v.cachedVIs[name] {
		return StatusOK
	}
	return StatusNotFound
}
