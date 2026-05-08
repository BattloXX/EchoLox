package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/discovery"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

// handleDiscoverAlexa returns all Echo/Alexa devices that have contacted this bridge via SSDP.
func (h *Handler) handleDiscoverAlexa(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	writeJSON(w, discovery.AllAlexas())
}

// handleDiscoverLoxone fetches echolox_* Virtual Inputs from the Miniserver
// and returns them grouped as importable device proposals.
func (h *Handler) handleDiscoverLoxone(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	if h.lox == nil {
		writeErr(w, 503, fmt.Errorf("kein Miniserver konfiguriert — bitte in Einstellungen prüfen"))
		return
	}

	proposals, err := loxone.FetchVIProposals(h.lox)
	if err != nil {
		writeErr(w, 500, err)
		return
	}

	// Mark proposals whose VIs already exist in the device store.
	existing := make(map[string]bool)
	for _, d := range h.mgr.All() {
		for _, vi := range d.VirtualInputs {
			existing[vi] = true
		}
	}

	type proposalResponse struct {
		loxone.VIProposal
		AlreadyExists bool `json:"already_exists"`
	}
	result := make([]proposalResponse, 0, len(proposals))
	for _, p := range proposals {
		alreadyExists := false
		for _, vi := range p.VIs {
			if existing[vi] {
				alreadyExists = true
				break
			}
		}
		result = append(result, proposalResponse{VIProposal: p, AlreadyExists: alreadyExists})
	}
	writeJSON(w, result)
}

// handleDiscoverLoxoneImport creates EchoLox devices from the selected VI proposals.
func (h *Handler) handleDiscoverLoxoneImport(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if h.lox == nil {
		writeErr(w, 503, fmt.Errorf("kein Miniserver konfiguriert"))
		return
	}

	var req struct {
		Bases []string `json:"bases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, 400, err)
		return
	}

	proposals, err := loxone.FetchVIProposals(h.lox)
	if err != nil {
		writeErr(w, 500, err)
		return
	}
	byBase := make(map[string]loxone.VIProposal, len(proposals))
	for _, p := range proposals {
		byBase[p.Base] = p
	}

	imported := []string{}
	errors := map[string]string{}
	for _, base := range req.Bases {
		p, ok := byBase[base]
		if !ok {
			errors[base] = "nicht auf Miniserver gefunden"
			continue
		}
		d := &device.Device{
			Name:       p.DisplayName,
			Type:       device.DeviceType(p.Type),
			SwitchMode: device.SwitchMode(p.SwitchMode),
			Transport:  "http",
		}
		if err := h.mgr.Create(d); err != nil {
			errors[base] = err.Error()
		} else {
			imported = append(imported, p.DisplayName)
		}
	}
	writeJSON(w, map[string]interface{}{
		"imported": imported,
		"errors":   errors,
	})
}
