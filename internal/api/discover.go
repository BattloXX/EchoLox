package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

	// Build existing name set to prevent re-importing already-present devices.
	existingNames := make(map[string]bool)
	for _, d := range h.mgr.All() {
		existingNames[device.NormalizeName(d.Name)] = true
	}

	imported := []string{}
	errors := map[string]string{}
	for _, base := range req.Bases {
		p, ok := byBase[base]
		if !ok {
			errors[base] = "nicht auf Miniserver gefunden"
			continue
		}
		norm := device.NormalizeName(p.DisplayName)
		if existingNames[norm] {
			errors[base] = "Gerät mit diesem Namen existiert bereits"
			continue
		}
		existingNames[norm] = true
		// Build VirtualInputs map.
		// Direct controls (Dimmer, Switch, LightControllerV2) have a single VI
		// equal to the base name — map all roles to that same name so Loxone
		// receives e.g. /dev/sps/io/WohnzimmerLicht/1 for on and /50 for brightness.
		vis := make(map[string]string, len(p.VIs))
		if len(p.VIs) == 1 && p.VIs[0] == p.Base {
			name := p.Base
			vis["on"] = name
			vis["off"] = name
			switch device.DeviceType(p.Type) {
			case device.TypeDimmer:
				vis["brightness"] = name
			case device.TypeColor:
				vis["brightness"] = name
				vis["hue"] = name
				vis["saturation"] = name
			}
		} else {
			for _, vi := range p.VIs {
				role := strings.TrimPrefix(vi, p.Base)
				role = strings.TrimPrefix(role, "_")
				if role == "" {
					role = "on"
				}
				vis[role] = vi
			}
		}
		d := &device.Device{
			Name:          p.DisplayName,
			Type:          device.DeviceType(p.Type),
			SwitchMode:    device.SwitchMode(p.SwitchMode),
			Transport:     "http",
			VirtualInputs: vis,
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
