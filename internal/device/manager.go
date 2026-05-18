package device

import (
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ErrDuplicateName is returned by Create and Update when a device with the same
// normalized name already exists.
var ErrDuplicateName = errors.New("duplicate device name")

var reUniqueID = regexp.MustCompile(`^00:17:88(:[0-9a-f]{2}){5}-[0-9a-f]{2}$`)

type Manager struct {
	mu      sync.RWMutex
	store   *Store
	byID    map[string]*Device // internal UUID → device
	byHueID map[string]*Device // Hue numeric ID → device
	states  map[string]*State  // internal UUID → state
	nextHue int                // next HueID counter

	// notifyFn is called by TriggerNotify to request a fresh SSDP NOTIFY burst.
	// Set by the bridge after construction; nil-safe.
	notifyFn func()
}

// uniqueIDFromDeviceID derives a stable Hue uniqueid from a device's internal ID.
// The result is deterministic and guaranteed different from the old HueID-based values
// (which started with 00:17:88:01:00:…). Using md5 of the stable Device.ID avoids
// any dependency on the reassignable HueID counter.
func uniqueIDFromDeviceID(id string) string {
	return uniqueIDFromDeviceIDSalt(id, 0)
}

func uniqueIDFromDeviceIDSalt(id string, salt int) string {
	h := md5.Sum([]byte(fmt.Sprintf("%s:%d", id, salt)))
	return fmt.Sprintf("00:17:88:%02x:%02x:%02x:%02x:%02x-%02x",
		h[0], h[1], h[2], h[3], h[4], h[5])
}

// SetNotifyFn registers the callback that triggers an SSDP NOTIFY burst.
func (m *Manager) SetNotifyFn(fn func()) {
	m.mu.Lock()
	m.notifyFn = fn
	m.mu.Unlock()
}

// TriggerNotify fires a fresh SSDP NOTIFY burst (no-op if not wired up).
func (m *Manager) TriggerNotify() {
	m.mu.RLock()
	fn := m.notifyFn
	m.mu.RUnlock()
	if fn != nil {
		go fn()
	}
}

func NewManager(dbPath string) (*Manager, error) {
	s := NewStore(dbPath)
	devices, err := s.Load()
	if err != nil {
		return nil, fmt.Errorf("load devices: %w", err)
	}
	m := &Manager{
		store:   s,
		byID:    make(map[string]*Device),
		byHueID: make(map[string]*Device),
		states:  make(map[string]*State),
		nextHue: 1,
	}
	changed := m.buildIndexes(devices)
	changed = m.selfHeal(devices) || changed
	if changed {
		if err := m.backupAndPersist(devices); err != nil {
			log.Printf("selfHeal: persist failed: %v", err)
		}
	}
	return m, nil
}

// buildIndexes populates byID, byHueID, states, and nextHue from the given slice.
// Must be called with m.mu not held (used during construction before anything is running).
func (m *Manager) buildIndexes(devices []*Device) (changed bool) {
	m.byID = make(map[string]*Device)
	m.byHueID = make(map[string]*Device)
	m.states = make(map[string]*State)
	m.nextHue = 1
	for _, d := range devices {
		if d.HueID == "" {
			d.HueID = strconv.Itoa(m.nextHue)
			changed = true
		}
		n, _ := strconv.Atoi(d.HueID)
		if n >= m.nextHue {
			m.nextHue = n + 1
		}
		m.byID[d.ID] = d
		m.byHueID[d.HueID] = d
		if m.states[d.ID] == nil {
			m.states[d.ID] = &State{Brightness: 254}
		}
	}
	return changed
}

// selfHeal checks every device for invariant violations and fixes them in-place.
// Returns true if anything was changed (caller must persist + backup).
func (m *Manager) selfHeal(devices []*Device) (changed bool) {
	seenUID := map[string]bool{}  // uniqueid → already used
	seenHue := map[string]bool{}  // hue_id → already used
	seenName := map[string]bool{} // normalized name → already used

	knownTypes := map[DeviceType]bool{
		TypeSwitch: true, TypeDimmer: true, TypeColor: true, TypeScene: true,
	}

	for _, d := range devices {
		// ── ID ───────────────────────────────────────────────────────────
		if d.ID == "" {
			d.ID = uuid.New().String()[:18]
			log.Printf("selfHeal: device %q: ID was empty, assigned %s", d.Name, d.ID)
			changed = true
		}

		// ── UniqueID ─────────────────────────────────────────────────────
		if !reUniqueID.MatchString(d.UniqueID) {
			d.UniqueID = uniqueIDFromDeviceID(d.ID)
			log.Printf("selfHeal: device %q: uniqueid generated from ID: %s", d.Name, d.UniqueID)
			changed = true
		}
		// Resolve collision by salting
		for salt := 1; seenUID[d.UniqueID]; salt++ {
			d.UniqueID = uniqueIDFromDeviceIDSalt(d.ID, salt)
			log.Printf("selfHeal: device %q: uniqueid collision resolved (salt=%d): %s", d.Name, salt, d.UniqueID)
			changed = true
		}
		seenUID[d.UniqueID] = true

		// ── HueID ────────────────────────────────────────────────────────
		if d.HueID == "" || seenHue[d.HueID] {
			d.HueID = strconv.Itoa(m.nextHue)
			m.nextHue++
			log.Printf("selfHeal: device %q: HueID reassigned to %s (duplicate or empty)", d.Name, d.HueID)
			changed = true
		} else {
			n, _ := strconv.Atoi(d.HueID)
			if n >= m.nextHue {
				m.nextHue = n + 1
			}
		}
		seenHue[d.HueID] = true

		// ── Type ─────────────────────────────────────────────────────────
		if !knownTypes[d.Type] {
			log.Printf("selfHeal: device %q: unknown type %q, defaulting to switch", d.Name, d.Type)
			d.Type = TypeSwitch
			changed = true
		}

		// ── Transport ────────────────────────────────────────────────────
		if d.Transport == "" {
			d.Transport = "http"
			log.Printf("selfHeal: device %q: transport was empty, defaulting to http", d.Name)
			changed = true
		}

		// ── LastSent ─────────────────────────────────────────────────────
		if d.LastSent == nil {
			d.LastSent = map[string]SentRecord{}
			changed = true
		}

		// ── VirtualInputs — regenerate when empty or prefix mismatched ───
		expected := GenerateVirtualInputs(d.Name, d.Type, d.SwitchMode)
		if len(d.VirtualInputs) == 0 || !virtualInputsMatch(d.VirtualInputs, expected) {
			d.VirtualInputs = expected
			log.Printf("selfHeal: device %q: virtual_inputs regenerated", d.Name)
			changed = true
		}

		// ── Duplicate name detection (warn only, do not delete) ──────────
		norm := NormalizeName(d.Name)
		if seenName[norm] {
			log.Printf("selfHeal: WARNING device %q: normalized name %q already seen — possible duplicate", d.Name, norm)
		}
		seenName[norm] = true
	}

	// Rebuild indexes after potential HueID / UniqueID changes
	if changed {
		m.buildIndexes(devices)
	}
	return changed
}

// virtualInputsMatch returns true when all keys/values in want are present in got.
func virtualInputsMatch(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

// backupAndPersist saves a .bak copy of the current devices.json, then persists devices.
func (m *Manager) backupAndPersist(devices []*Device) error {
	// Best-effort backup of the original file
	if data, err := os.ReadFile(m.store.path); err == nil {
		_ = os.WriteFile(m.store.path+".bak", data, 0644)
	}
	return m.store.Save(devices)
}

// All returns all devices sorted by HueID (stable order for Hue API).
func (m *Manager) All() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Device, 0, len(m.byID))
	for _, d := range m.byID {
		list = append(list, d)
	}
	sort.Slice(list, func(i, j int) bool {
		a, _ := strconv.Atoi(list[i].HueID)
		b, _ := strconv.Atoi(list[j].HueID)
		return a < b
	})
	return list
}

func (m *Manager) Get(id string) (*Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.byID[id]
	return d, ok
}

// GetByHueID looks up a device by its short Hue numeric ID.
func (m *Manager) GetByHueID(hueID string) (*Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.byHueID[hueID]
	return d, ok
}

func (m *Manager) GetState(id string) *State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.states[id]; ok {
		cp := *s
		return &cp
	}
	return &State{Brightness: 254}
}

func (m *Manager) SetState(id string, s *State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[id] = s
}

func (m *Manager) Create(d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.ID == "" {
		d.ID = uuid.New().String()[:18]
	}
	if m.nameInUse(NormalizeName(d.Name), "") {
		return fmt.Errorf("%w: Ein Gerät mit dem Namen %q existiert bereits", ErrDuplicateName, d.Name)
	}
	if d.HueID == "" {
		d.HueID = strconv.Itoa(m.nextHue)
		m.nextHue++
	}
	// Assign stable uniqueid derived from the device's own ID, not the HueID counter.
	if !reUniqueID.MatchString(d.UniqueID) {
		d.UniqueID = uniqueIDFromDeviceID(d.ID)
		// Resolve any collision with existing uniqueids
		for salt := 1; m.uniqueIDInUse(d.UniqueID); salt++ {
			d.UniqueID = uniqueIDFromDeviceIDSalt(d.ID, salt)
		}
	}
	if len(d.VirtualInputs) == 0 {
		d.VirtualInputs = GenerateVirtualInputs(d.Name, d.Type, d.SwitchMode)
	}
	if d.LastSent == nil {
		d.LastSent = map[string]SentRecord{}
	}
	if d.Transport == "" {
		d.Transport = "http"
	}
	m.byID[d.ID] = d
	m.byHueID[d.HueID] = d
	m.states[d.ID] = &State{Brightness: 254}
	return m.persist()
}

// uniqueIDInUse checks whether the given uniqueid is already assigned to any device.
// Caller must hold m.mu (or be in a phase where no concurrent access is possible).
func (m *Manager) uniqueIDInUse(uid string) bool {
	for _, d := range m.byID {
		if d.UniqueID == uid {
			return true
		}
	}
	return false
}

// nameInUse returns true if any device (other than excludeID) has the same
// normalized name. Caller must hold m.mu.
func (m *Manager) nameInUse(norm, excludeID string) bool {
	for id, d := range m.byID {
		if id != excludeID && NormalizeName(d.Name) == norm {
			return true
		}
	}
	return false
}

func (m *Manager) Update(d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nameInUse(NormalizeName(d.Name), d.ID) {
		return fmt.Errorf("%w: Ein Gerät mit dem Namen %q existiert bereits", ErrDuplicateName, d.Name)
	}
	if old, ok := m.byID[d.ID]; ok {
		if d.HueID == "" {
			d.HueID = old.HueID
		}
		// UniqueID is immutable — always carry over from the existing record.
		d.UniqueID = old.UniqueID
		// Carry over LastSent so history survives an edit
		if d.LastSent == nil {
			d.LastSent = old.LastSent
		}
	}
	// Always regenerate VIs to stay in sync with name/type/switchMode changes
	d.VirtualInputs = GenerateVirtualInputs(d.Name, d.Type, d.SwitchMode)
	m.byID[d.ID] = d
	m.byHueID[d.HueID] = d
	return m.persist()
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d, ok := m.byID[id]; ok {
		delete(m.byHueID, d.HueID)
	}
	delete(m.byID, id)
	delete(m.states, id)
	return m.persist()
}

func (m *Manager) RecordSent(id, input, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.byID[id]
	if !ok {
		return
	}
	if d.LastSent == nil {
		d.LastSent = map[string]SentRecord{}
	}
	d.LastSent[input] = SentRecord{Value: value, At: time.Now().UTC()}
	_ = m.persist()
}

func (m *Manager) persist() error {
	list := make([]*Device, 0, len(m.byID))
	for _, d := range m.byID {
		list = append(list, d)
	}
	return m.store.Save(list)
}

// DBPath returns the path to the devices.json file.
func (m *Manager) DBPath() string { return m.store.path }

// Reload re-reads devices.json from disk, runs selfHeal, and replaces in-memory state.
func (m *Manager) Reload() error {
	devices, err := m.store.Load()
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID = make(map[string]*Device)
	m.byHueID = make(map[string]*Device)
	m.states = make(map[string]*State)
	m.nextHue = 1
	changed := m.buildIndexes(devices)
	changed = m.selfHeal(devices) || changed
	if changed {
		if err := m.backupAndPersist(devices); err != nil {
			log.Printf("selfHeal (reload): persist failed: %v", err)
		}
	}
	return nil
}
