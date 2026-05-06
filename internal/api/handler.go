package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

type Handler struct {
	mgr      *device.Manager
	lox      *loxone.Client
	verifier *loxone.Verifier
	lbs      map[string]loxone.LBMiniserver
}

func NewHandler(mgr *device.Manager, lox *loxone.Client, verifier *loxone.Verifier, lbs map[string]loxone.LBMiniserver) *Handler {
	return &Handler{mgr: mgr, lox: lox, verifier: verifier, lbs: lbs}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/echolox/api/devices", h.handleDevices)
	mux.HandleFunc("/echolox/api/devices/", h.handleDevice)
	mux.HandleFunc("/echolox/api/status", h.handleStatus)
	mux.HandleFunc("/echolox/api/miniservers", h.handleMiniservers)
	mux.HandleFunc("/echolox/api/verify", h.handleVerify)
}

func (h *Handler) handleDevices(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, h.mgr.All())
	case http.MethodPost:
		var d device.Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := h.mgr.Create(&d); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, d)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (h *Handler) handleDevice(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	id := strings.TrimPrefix(r.URL.Path, "/echolox/api/devices/")
	id = strings.TrimSuffix(id, "/test")
	isTest := strings.HasSuffix(r.URL.Path, "/test")

	if isTest && r.Method == http.MethodPost {
		h.handleTest(w, r, id)
		return
	}

	d, ok := h.mgr.Get(id)
	if !ok {
		http.Error(w, "not found", 404)
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, d)
	case http.MethodPut:
		if err := json.NewDecoder(r.Body).Decode(d); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		d.ID = id
		if err := h.mgr.Update(d); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, d)
	case http.MethodDelete:
		if err := h.mgr.Delete(id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]string{"deleted": id})
	}
}

func (h *Handler) handleTest(w http.ResponseWriter, r *http.Request, id string) {
	d, ok := h.mgr.Get(id)
	if !ok {
		http.Error(w, "not found", 404)
		return
	}
	if h.lox == nil {
		writeJSON(w, map[string]string{"status": "no loxone client"})
		return
	}
	errors := map[string]string{}
	for key, vi := range d.VirtualInputs {
		val := "1"
		if key == "brightness" {
			val = "50"
		}
		if err := h.lox.Send(vi, val); err != nil {
			errors[vi] = err.Error()
		} else {
			h.mgr.RecordSent(id, vi, val)
		}
	}
	if len(errors) > 0 {
		writeJSON(w, map[string]interface{}{"status": "partial", "errors": errors})
	} else {
		writeJSON(w, map[string]string{"status": "ok"})
	}
}

type viStatusRow struct {
	Name       string          `json:"name"`
	DeviceID   string          `json:"device_id"`
	DeviceName string          `json:"device_name"`
	Key        string          `json:"key"`
	Status     loxone.VIStatus `json:"status"`
	LastValue  string          `json:"last_value,omitempty"`
	LastSent   string          `json:"last_sent,omitempty"`
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodPost {
		// Refresh verify cache
		if err := h.verifier.RefreshCache(); err != nil {
			writeJSON(w, map[string]string{"error": err.Error()})
			return
		}
	}
	rows := []viStatusRow{}
	for _, d := range h.mgr.All() {
		for key, vi := range d.VirtualInputs {
			row := viStatusRow{
				Name:       vi,
				DeviceID:   d.ID,
				DeviceName: d.Name,
				Key:        key,
			}
			if d.LastSent != nil {
				if rec, ok := d.LastSent[vi]; ok {
					row.LastValue = rec.Value
					row.LastSent = rec.At.Format("02.01. 15:04:05")
					row.Status = h.verifier.CheckVI(vi)
				} else {
					row.Status = loxone.StatusNotSent
				}
			} else {
				row.Status = h.verifier.CheckVI(vi)
			}
			rows = append(rows, row)
		}
	}
	writeJSON(w, rows)
}

func (h *Handler) handleMiniservers(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	result := []map[string]string{}
	for id, ms := range h.lbs {
		result = append(result, map[string]string{
			"id":   id,
			"name": ms.Name,
			"ip":   ms.IPAddress,
		})
	}
	writeJSON(w, result)
}

func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if err := h.verifier.RefreshCache(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
