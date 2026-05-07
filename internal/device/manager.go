package device

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	mu       sync.RWMutex
	store    *Store
	byID     map[string]*Device  // internal UUID → device
	byHueID  map[string]*Device  // Hue numeric ID → device
	states   map[string]*State   // internal UUID → state
	nextHue  int                 // next HueID counter
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
	for _, d := range devices {
		// Assign HueID to devices that were saved before this field existed
		if d.HueID == "" {
			d.HueID = strconv.Itoa(m.nextHue)
		}
		n, _ := strconv.Atoi(d.HueID)
		if n >= m.nextHue {
			m.nextHue = n + 1
		}
		m.byID[d.ID] = d
		m.byHueID[d.HueID] = d
		m.states[d.ID] = &State{Brightness: 254}
	}
	return m, nil
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
	if d.HueID == "" {
		d.HueID = strconv.Itoa(m.nextHue)
		m.nextHue++
	}
	if len(d.VirtualInputs) == 0 {
		d.VirtualInputs = GenerateVirtualInputs(d.Name, d.Type, d.SwitchMode)
	}
	if d.LastSent == nil {
		d.LastSent = map[string]SentRecord{}
	}
	m.byID[d.ID] = d
	m.byHueID[d.HueID] = d
	m.states[d.ID] = &State{Brightness: 254}
	return m.persist()
}

func (m *Manager) Update(d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Keep old HueID if not supplied
	if old, ok := m.byID[d.ID]; ok && d.HueID == "" {
		d.HueID = old.HueID
	}
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

// Reload re-reads devices.json from disk, replacing the in-memory state.
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
	for _, d := range devices {
		if d.HueID == "" {
			d.HueID = strconv.Itoa(m.nextHue)
		}
		n, _ := strconv.Atoi(d.HueID)
		if n >= m.nextHue {
			m.nextHue = n + 1
		}
		m.byID[d.ID] = d
		m.byHueID[d.HueID] = d
		m.states[d.ID] = &State{Brightness: 254}
	}
	return nil
}
