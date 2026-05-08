package discovery

import (
	"sort"
	"sync"
	"time"
)

type AlexaDevice struct {
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	LastSeen  time.Time `json:"last_seen"`
}

var (
	mu      sync.RWMutex
	devices = map[string]*AlexaDevice{}
)

func RecordAlexa(ip, ua string) {
	mu.Lock()
	defer mu.Unlock()
	if e, ok := devices[ip]; ok {
		e.LastSeen = time.Now()
		if ua != "" {
			e.UserAgent = ua
		}
	} else {
		devices[ip] = &AlexaDevice{IP: ip, UserAgent: ua, LastSeen: time.Now()}
	}
}

func AllAlexas() []AlexaDevice {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]AlexaDevice, 0, len(devices))
	for _, e := range devices {
		result = append(result, *e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})
	return result
}
