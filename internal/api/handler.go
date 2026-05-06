package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

type Handler struct {
	mgr      *device.Manager
	lox      *loxone.Client
	verifier *loxone.Verifier
	lbs      map[string]loxone.LBMiniserver
	cfgPath  string
}

func NewHandler(mgr *device.Manager, lox *loxone.Client, verifier *loxone.Verifier, lbs map[string]loxone.LBMiniserver, cfgPath string) *Handler {
	return &Handler{mgr: mgr, lox: lox, verifier: verifier, lbs: lbs, cfgPath: cfgPath}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/echolox/api/devices", h.handleDevices)
	mux.HandleFunc("/echolox/api/devices/", h.handleDevice)
	mux.HandleFunc("/echolox/api/status", h.handleStatus)
	mux.HandleFunc("/echolox/api/miniservers", h.handleMiniservers)
	mux.HandleFunc("/echolox/api/verify", h.handleVerify)
	mux.HandleFunc("/echolox/api/settings", h.handleSettings)
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

// configYAML mirrors bridge.Config for reading/writing without an import cycle.
type configYAML struct {
	Server struct {
		Port int    `yaml:"port"`
		IP   string `yaml:"ip,omitempty"`
	} `yaml:"server"`
	UPNP struct {
		Name string `yaml:"name,omitempty"`
		UUID string `yaml:"uuid,omitempty"`
	} `yaml:"upnp,omitempty"`
	Loxone struct {
		Miniserver string `yaml:"miniserver"`
		Transport  string `yaml:"transport"`
		UDPPort    int    `yaml:"udp_port"`
	} `yaml:"loxone"`
	MQTT struct {
		Broker   string `yaml:"broker,omitempty"`
		Username string `yaml:"username,omitempty"`
		Password string `yaml:"password,omitempty"`
	} `yaml:"mqtt,omitempty"`
	DataDir string `yaml:"data_dir,omitempty"`
}

func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == http.MethodOptions {
		return
	}

	readCfg := func() configYAML {
		cfg := configYAML{}
		cfg.Server.Port = 8079
		cfg.Loxone.Miniserver = "1"
		cfg.Loxone.Transport = "http"
		cfg.Loxone.UDPPort = 7777
		cfg.MQTT.Broker = "tcp://localhost:1883"
		if h.cfgPath != "" {
			if data, err := os.ReadFile(h.cfgPath); err == nil {
				yaml.Unmarshal(data, &cfg)
			}
		}
		return cfg
	}

	switch r.Method {
	case http.MethodGet:
		cfg := readCfg()
		writeJSON(w, map[string]interface{}{
			"miniserver":  cfg.Loxone.Miniserver,
			"transport":   cfg.Loxone.Transport,
			"udp_port":    cfg.Loxone.UDPPort,
			"port":        cfg.Server.Port,
			"mqtt_broker": cfg.MQTT.Broker,
		})

	case http.MethodPost:
		var req struct {
			Miniserver string `json:"miniserver"`
			Transport  string `json:"transport"`
			UDPPort    int    `json:"udp_port"`
			Port       int    `json:"port"`
			MQTTBroker string `json:"mqtt_broker"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		cfg := readCfg()
		if req.Miniserver != "" {
			cfg.Loxone.Miniserver = req.Miniserver
		}
		if req.Transport != "" {
			cfg.Loxone.Transport = req.Transport
		}
		if req.UDPPort > 0 {
			cfg.Loxone.UDPPort = req.UDPPort
		}
		if req.Port > 0 {
			cfg.Server.Port = req.Port
		}
		if req.MQTTBroker != "" {
			cfg.MQTT.Broker = req.MQTTBroker
		}
		if h.cfgPath == "" {
			http.Error(w, "no config path configured", 500)
			return
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := os.WriteFile(h.cfgPath, data, 0644); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, map[string]string{
			"status":  "ok",
			"message": "Gespeichert — EchoLox neu starten um Portänderungen zu übernehmen",
		})

	default:
		http.Error(w, "method not allowed", 405)
	}
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
