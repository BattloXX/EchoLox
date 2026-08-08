// Package lbloglevel reads and caches EchoLox's LoxBerry Plugin Manager log level.
package lbloglevel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const refreshInterval = 30 * time.Second

type pluginDatabase struct {
	Plugins map[string]struct {
		Folder   string `json:"folder"`
		LogLevel *int   `json:"loglevel"`
	} `json:"plugins"`
}

type reader struct {
	mu      sync.RWMutex
	path    string
	level   int
	modTime time.Time
	exists  bool
}

func newReader(path string) *reader {
	return &reader{path: path, level: -1}
}

func databasePath() string {
	lbHome := os.Getenv("LBHOMEDIR")
	if lbHome == "" {
		lbHome = "/opt/loxberry"
	}
	return filepath.Join(lbHome, "data", "system", "plugindatabase.json")
}

var (
	globalMu sync.Mutex
	global   *reader
	started  bool
)

func globalReader() *reader {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = newReader(databasePath())
	}
	return global
}

// Start loads the current value and starts an idempotent background refresh.
func Start() {
	globalMu.Lock()
	if started {
		globalMu.Unlock()
		return
	}
	started = true
	if global == nil {
		global = newReader(databasePath())
	}
	r := global
	globalMu.Unlock()

	r.refresh()
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			r.refresh()
		}
	}()
}

// CurrentLevel returns the cached LoxBerry log level. Missing and disabled values return -1.
func CurrentLevel() int {
	return globalReader().currentLevel()
}

// ShouldForward applies a deliberately simple two-way mapping because EchoLox
// currently has only Info and Debug severities, rather than LoxBerry's full scale.
func ShouldForward(debug bool) bool {
	return shouldForward(CurrentLevel(), debug)
}

func shouldForward(level int, debug bool) bool {
	if level < 1 || level > 7 {
		return false
	}
	return !debug || level == 7
}

func (r *reader) currentLevel() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.level
}

func (r *reader) refresh() {
	info, err := os.Stat(r.path)
	if err != nil {
		r.setUnavailable()
		return
	}

	r.mu.RLock()
	unchanged := r.exists && info.ModTime().Equal(r.modTime)
	r.mu.RUnlock()
	if unchanged {
		return
	}

	data, err := os.ReadFile(r.path)
	if err != nil {
		r.setUnavailable()
		return
	}
	var db pluginDatabase
	if err := json.Unmarshal(data, &db); err != nil {
		r.setUnavailable()
		return
	}

	level := -1
	for _, plugin := range db.Plugins {
		if plugin.Folder == "EchoLox" && plugin.LogLevel != nil {
			level = *plugin.LogLevel
			break
		}
	}
	r.mu.Lock()
	r.level = level
	r.modTime = info.ModTime()
	r.exists = true
	r.mu.Unlock()
}

func (r *reader) setUnavailable() {
	r.mu.Lock()
	r.level = -1
	r.exists = false
	r.modTime = time.Time{}
	r.mu.Unlock()
}
