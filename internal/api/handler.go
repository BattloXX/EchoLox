package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/logbuf"
	"github.com/BattloXX/EchoLox/internal/loxone"
)

type Handler struct {
	mgr      *device.Manager
	lox      *loxone.Client
	verifier *loxone.Verifier
	lbs      map[string]loxone.LBMiniserver
	cfgPath  string
	dataDir  string
}

func NewHandler(mgr *device.Manager, lox *loxone.Client, verifier *loxone.Verifier, lbs map[string]loxone.LBMiniserver, cfgPath, dataDir string) *Handler {
	return &Handler{mgr: mgr, lox: lox, verifier: verifier, lbs: lbs, cfgPath: cfgPath, dataDir: dataDir}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/echolox/api/devices", h.handleDevices)
	mux.HandleFunc("/echolox/api/devices/", h.handleDevice)
	mux.HandleFunc("/echolox/api/status", h.handleStatus)
	mux.HandleFunc("/echolox/api/miniservers", h.handleMiniservers)
	mux.HandleFunc("/echolox/api/verify", h.handleVerify)
	mux.HandleFunc("/echolox/api/settings", h.handleSettings)
	mux.HandleFunc("/echolox/api/restart", h.handleRestart)
	mux.HandleFunc("/echolox/api/backup/download", h.handleBackupDownload)
	mux.HandleFunc("/echolox/api/backup/restore", h.handleBackupRestore)
	mux.HandleFunc("/echolox/api/backup/restore-local", h.handleBackupRestoreLocal)
	mux.HandleFunc("/echolox/api/backup/delete", h.handleBackupDelete)
	mux.HandleFunc("/echolox/api/backup", h.handleBackup)
	mux.HandleFunc("/echolox/api/logs/download", h.handleLogsDownload)
	mux.HandleFunc("/echolox/api/logs/level", h.handleLogsLevel)
	mux.HandleFunc("/echolox/api/logs", h.handleLogs)
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
		Port          int    `yaml:"port"`
		IP            string `yaml:"ip,omitempty"`
		DiscoveryPort int    `yaml:"discovery_port,omitempty"`
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
			"miniserver":     cfg.Loxone.Miniserver,
			"transport":      cfg.Loxone.Transport,
			"udp_port":       cfg.Loxone.UDPPort,
			"port":           cfg.Server.Port,
			"discovery_port": cfg.Server.DiscoveryPort,
			"mqtt_broker":    cfg.MQTT.Broker,
		})

	case http.MethodPost:
		var req struct {
			Miniserver    string `json:"miniserver"`
			Transport     string `json:"transport"`
			UDPPort       int    `json:"udp_port"`
			Port          int    `json:"port"`
			DiscoveryPort int    `json:"discovery_port"`
			MQTTBroker    string `json:"mqtt_broker"`
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
		cfg.Server.DiscoveryPort = req.DiscoveryPort
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

// ── Restart ────────────────────────────────────────────────────────────────

func (h *Handler) handleRestart(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	writeJSON(w, map[string]string{"status": "ok", "message": "EchoLox wird neu gestartet…"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(600 * time.Millisecond)
		if err := exec.Command("systemctl", "restart", "echolox.service").Run(); err != nil {
			os.Exit(0) // let systemd restart via Restart=on-failure
		}
	}()
}

// ── Backup ─────────────────────────────────────────────────────────────────

func (h *Handler) handleBackup(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	switch r.Method {
	case http.MethodGet:
		h.backupList(w)
	case http.MethodPost:
		h.backupCreateLocal(w)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (h *Handler) backupList(w http.ResponseWriter) {
	backupDir := filepath.Join(h.dataDir, "backup")
	entries, _ := os.ReadDir(backupDir)
	type item struct {
		File      string `json:"file"`
		Timestamp string `json:"timestamp"`
		Display   string `json:"display"`
	}
	result := []item{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".zip") {
			continue
		}
		name := e.Name() // echolox_backup_20060102_150405.zip
		if len(name) < 34 {
			continue
		}
		ts := name[len(name)-19 : len(name)-4] // 20060102_150405
		t, err := time.ParseInLocation("20060102_150405", ts, time.Local)
		display := ts
		if err == nil {
			display = t.Format("02.01.2006 15:04:05")
		}
		result = append(result, item{File: name, Timestamp: ts, Display: display})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Timestamp > result[j].Timestamp })
	writeJSON(w, result)
}

func (h *Handler) backupCreateLocal(w http.ResponseWriter) {
	backupDir := filepath.Join(h.dataDir, "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ts := time.Now().Format("20060102_150405")
	zipPath := filepath.Join(backupDir, "echolox_backup_"+ts+".zip")
	if err := h.writeBackupZip(zipPath); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"status": "ok", "timestamp": ts, "file": filepath.Base(zipPath)})
}

func (h *Handler) writeBackupZip(dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	addToZip := func(src, name string) {
		data, err := os.ReadFile(src)
		if err != nil {
			return
		}
		fw, err := zw.Create(name)
		if err != nil {
			return
		}
		fw.Write(data)
	}
	if h.cfgPath != "" {
		addToZip(h.cfgPath, "EchoLox.cfg")
	}
	addToZip(filepath.Join(h.dataDir, "devices.json"), "devices.json")
	return nil
}

func (h *Handler) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	addToZip := func(src, name string) {
		if data, err := os.ReadFile(src); err == nil {
			if fw, err := zw.Create(name); err == nil {
				fw.Write(data)
			}
		}
	}
	if h.cfgPath != "" {
		addToZip(h.cfgPath, "EchoLox.cfg")
	}
	addToZip(filepath.Join(h.dataDir, "devices.json"), "devices.json")
	zw.Close()
	ts := time.Now().Format("20060102_150405")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="echolox_backup_%s.zip"`, ts))
	w.Write(buf.Bytes())
}

func (h *Handler) handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "backup-Datei erforderlich: "+err.Error(), 400)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	h.restoreFromZipBytes(w, data)
}

func (h *Handler) handleBackupRestoreLocal(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.File == "" {
		http.Error(w, "file erforderlich", 400)
		return
	}
	// Prevent path traversal
	if strings.Contains(req.File, "/") || strings.Contains(req.File, "\\") {
		http.Error(w, "ungültiger Dateiname", 400)
		return
	}
	zipPath := filepath.Join(h.dataDir, "backup", req.File)
	data, err := os.ReadFile(zipPath)
	if err != nil {
		http.Error(w, "Datei nicht gefunden: "+err.Error(), 404)
		return
	}
	h.restoreFromZipBytes(w, data)
}

func (h *Handler) handleBackupDelete(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		File string `json:"file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.File == "" {
		http.Error(w, "file erforderlich", 400)
		return
	}
	if strings.Contains(req.File, "/") || strings.Contains(req.File, "\\") {
		http.Error(w, "ungültiger Dateiname", 400)
		return
	}
	zipPath := filepath.Join(h.dataDir, "backup", req.File)
	if err := os.Remove(zipPath); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) restoreFromZipBytes(w http.ResponseWriter, data []byte) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		http.Error(w, "ungültige ZIP-Datei: "+err.Error(), 400)
		return
	}
	restored := []string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		switch f.Name {
		case "EchoLox.cfg":
			if h.cfgPath != "" {
				if os.WriteFile(h.cfgPath, content, 0644) == nil {
					restored = append(restored, "EchoLox.cfg")
				}
			}
		case "devices.json":
			devPath := filepath.Join(h.dataDir, "devices.json")
			if os.WriteFile(devPath, content, 0644) == nil {
				h.mgr.Reload()
				restored = append(restored, "devices.json")
			}
		}
	}
	if len(restored) == 0 {
		http.Error(w, "keine bekannten Dateien im Backup gefunden", 400)
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "restored": restored})
}

// ── Logs ───────────────────────────────────────────────────────────────────

func (h *Handler) handleLogs(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	entries := logbuf.Global.Entries()
	type entry struct {
		T     string `json:"t"`
		Level string `json:"level"`
		Msg   string `json:"msg"`
	}
	result := make([]entry, len(entries))
	for i, e := range entries {
		result[i] = entry{T: e.T.Format("2006-01-02 15:04:05"), Level: e.Level, Msg: e.Msg}
	}
	lvl := "info"
	if logbuf.Global.GetLevel() == logbuf.LevelDebug {
		lvl = "debug"
	}
	writeJSON(w, map[string]interface{}{"level": lvl, "entries": result})
}

func (h *Handler) handleLogsLevel(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	switch strings.ToLower(req.Level) {
	case "debug":
		logbuf.Global.SetLevel(logbuf.LevelDebug)
		logbuf.Global.Info("Log level set to DEBUG")
	default:
		logbuf.Global.SetLevel(logbuf.LevelInfo)
		logbuf.Global.Info("Log level set to INFO")
	}
	lvl := "info"
	if logbuf.Global.GetLevel() == logbuf.LevelDebug {
		lvl = "debug"
	}
	writeJSON(w, map[string]string{"status": "ok", "level": lvl})
}

func (h *Handler) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	entries := logbuf.Global.Entries()
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(e.T.Format("2006-01-02 15:04:05"))
		sb.WriteString(" [")
		sb.WriteString(e.Level)
		sb.WriteString("] ")
		sb.WriteString(e.Msg)
		sb.WriteByte('\n')
	}
	ts := time.Now().Format("20060102_150405")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="echolox_%s.log"`, ts))
	w.Write([]byte(sb.String()))
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
