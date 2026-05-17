package migrate

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BattloXX/EchoLox/internal/device"
)

// OldDevice represents the legacy ha-bridge device format
type OldDevice struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	MapType    string `json:"mapType"`
	DeviceType string `json:"deviceType"`
	OnUrl      string `json:"onUrl"`
	OffUrl     string `json:"offUrl"`
	DimUrl     string `json:"dimUrl"`
	ColorUrl   string `json:"colorUrl"`
}

type ImportResult struct {
	Imported []device.Device `json:"imported"`
	Skipped  []SkippedDevice `json:"skipped"`
}

type SkippedDevice struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

var skippedTypes = map[string]bool{
	"veraDevice":       true,
	"harmonyDevice":    true,
	"nestDevice":       true,
	"fibaroDevice":     true,
	"hassDevice":       true,
	"lifxDevice":       true,
	"mqttHADevice":     true,
	"domoticzDevice":   true,
	"fhemDevice":       true,
	"halDevice":        true,
	"openhabDevice":    true,
	"homewizardDevice": true,
	"broadlinkDevice":  true,
	"somfyDevice":      true,
}

func ParseOldDevices(data []byte) ([]OldDevice, error) {
	var devices []OldDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func MapDevice(old OldDevice) (*device.Device, string) {
	// Skip platform-specific types
	if skippedTypes[old.MapType] || skippedTypes[old.DeviceType] {
		return nil, fmt.Sprintf("platform-specific type: %s", old.MapType)
	}
	// Determine type
	dtype := device.TypeSwitch
	if old.ColorUrl != "" {
		dtype = device.TypeColor
	} else if old.DimUrl != "" {
		dtype = device.TypeDimmer
	}

	d := &device.Device{
		Name:          old.Name,
		Type:          dtype,
		VirtualInputs: device.GenerateVirtualInputs(old.Name, dtype, device.SwitchModeOnOff),
		Transport:     "http",
	}
	return d, ""
}

func ImportCLI(fromPath, toPath string) error {
	data, err := os.ReadFile(fromPath)
	if err != nil {
		return err
	}
	oldDevices, err := ParseOldDevices(data)
	if err != nil {
		return err
	}
	store := device.NewStore(toPath)
	existing, _ := store.Load()
	existingNames := existingNormNames(existing)
	for _, old := range oldDevices {
		d, reason := MapDevice(old)
		if d == nil {
			log.Printf("skip %s: %s", old.Name, reason)
			continue
		}
		norm := device.NormalizeName(d.Name)
		if existingNames[norm] {
			log.Printf("skip %s: device with normalized name %q already exists", old.Name, norm)
			continue
		}
		existingNames[norm] = true
		existing = append(existing, d)
	}
	return store.Save(existing)
}

// existingNormNames returns a set of normalized names for deduplication.
func existingNormNames(devices []*device.Device) map[string]bool {
	m := make(map[string]bool, len(devices))
	for _, d := range devices {
		m[device.NormalizeName(d.Name)] = true
	}
	return m
}

type Handler struct {
	mgr *device.Manager
}

func NewHandler(mgr *device.Manager) *Handler {
	return &Handler{mgr: mgr}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/echolox/api/import/preview", h.handlePreview)
	mux.HandleFunc("/echolox/api/import/confirm", h.handleConfirm)
}

func (h *Handler) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", 400)
		return
	}
	defer file.Close()

	_ = strings.Builder{}
	raw := make([]byte, 10<<20)
	n, _ := file.Read(raw)

	oldDevices, err := ParseOldDevices(raw[:n])
	if err != nil {
		http.Error(w, "parse error: "+err.Error(), 400)
		return
	}

	result := ImportResult{}
	for _, old := range oldDevices {
		d, reason := MapDevice(old)
		if d == nil {
			result.Skipped = append(result.Skipped, SkippedDevice{Name: old.Name, Reason: reason})
		} else {
			result.Imported = append(result.Imported, *d)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var devices []device.Device
	if err := json.NewDecoder(r.Body).Decode(&devices); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Build existing name set for dedup before creating any device.
	existingNames := make(map[string]bool)
	for _, d := range h.mgr.All() {
		existingNames[device.NormalizeName(d.Name)] = true
	}
	imported := 0
	skipped := 0
	for i := range devices {
		norm := device.NormalizeName(devices[i].Name)
		if existingNames[norm] {
			skipped++
			continue
		}
		existingNames[norm] = true
		if err := h.mgr.Create(&devices[i]); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		imported++
	}
	_ = filepath.Base(".")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"imported": imported, "skipped": skipped})
}
