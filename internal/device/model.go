package device

import "time"

type DeviceType string

const (
	TypeSwitch DeviceType = "switch"
	TypeDimmer DeviceType = "dimmer"
	TypeColor  DeviceType = "color"
	TypeScene  DeviceType = "scene"
)

type SwitchMode string

const (
	SwitchModeOnOff   SwitchMode = "onoff"
	SwitchModeImpulse SwitchMode = "impulse"
)

type SentRecord struct {
	Value string    `json:"value"`
	At    time.Time `json:"at"`
}

type Device struct {
	ID string `json:"id"`
	// HueID is the short numeric string exposed in the Hue API ("1", "2", ...).
	// Alexa requires stable numeric-like IDs across restarts.
	HueID string `json:"hue_id"`
	// UniqueID is the Hue MAC-format uniqueid (9 octets, "-" separator on last byte).
	// Derived once from the stable Device.ID and persisted — never changes after creation,
	// even if HueID is reassigned. Alexa keys its device cache by uniqueid; stability here
	// prevents stale-cache collisions when devices are recreated with the same HueID counter.
	UniqueID      string                `json:"unique_id,omitempty"`
	Name          string                `json:"name"`
	Type          DeviceType            `json:"type"`
	SwitchMode    SwitchMode            `json:"switch_mode,omitempty"`
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
