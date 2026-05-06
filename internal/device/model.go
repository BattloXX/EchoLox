package device

import "time"

type DeviceType string

const (
	TypeSwitch DeviceType = "switch"
	TypeDimmer DeviceType = "dimmer"
	TypeColor  DeviceType = "color"
	TypeScene  DeviceType = "scene"
)

type SentRecord struct {
	Value string    `json:"value"`
	At    time.Time `json:"at"`
}

type Device struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Type          DeviceType            `json:"type"`
	VirtualInputs map[string]string     `json:"virtual_inputs"`
	Transport     string                `json:"transport"`
	LastSent      map[string]SentRecord `json:"last_sent,omitempty"`
}

// State is in-memory only, not persisted
type State struct {
	On         bool
	Brightness int // 0-254
	Hue        int // 0-65535
	Saturation int // 0-254
	ColorTemp  int
	ColorMode  string
}
