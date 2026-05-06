package device

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	mu     sync.RWMutex
	store  *Store
	byID   map[string]*Device
	states map[string]*State
}

func NewManager(dbPath string) (*Manager, error) {
	s := NewStore(dbPath)
	devices, err := s.Load()
	if err != nil {
		return nil, fmt.Errorf("load devices: %w", err)
	}
	m := &Manager{
		store:  s,
		byID:   make(map[string]*Device),
		states: make(map[string]*State),
	}
	for _, d := range devices {
		m.byID[d.ID] = d
		m.states[d.ID] = &State{Brightness: 254}
	}
	return m, nil
}

func (m *Manager) All() []*Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Device, 0, len(m.byID))
	for _, d := range m.byID {
		list = append(list, d)
	}
	return list
}

func (m *Manager) Get(id string) (*Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.byID[id]
	return d, ok
}

func (m *Manager) GetState(id string) *State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.states[id]; ok {
		return s
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
	if len(d.VirtualInputs) == 0 {
		d.VirtualInputs = GenerateVirtualInputs(d.Name, d.Type)
	}
	if d.LastSent == nil {
		d.LastSent = map[string]SentRecord{}
	}
	m.byID[d.ID] = d
	m.states[d.ID] = &State{Brightness: 254}
	return m.persist()
}

func (m *Manager) Update(d *Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[d.ID] = d
	return m.persist()
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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
