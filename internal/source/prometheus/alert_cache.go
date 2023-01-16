package prometheus

import (
	"sync"
	"time"

	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
)

type alert struct {
	item         promApi.Alert
	lastAccessed int64
}

// AlertCache caches prometheus alerts to prevent recurring notification to Botkube
type AlertCache struct {
	l   sync.Mutex
	m   map[string]*alert
	TTL int64
}

// AlertCacheConfig alert cache configuration
type AlertCacheConfig struct {
	// TTL Time-to-live value for cache entry
	TTL int64
}

// NewAlertCache initializes new alert cache store
func NewAlertCache(config AlertCacheConfig) *AlertCache {
	alerts := AlertCache{
		m: make(map[string]*alert),
	}
	go func() {
		for now := range time.Tick(time.Second) {
			alerts.l.Lock()
			for k, v := range alerts.m {
				if now.Unix()-v.lastAccessed > config.TTL {
					delete(alerts.m, k)
				}
			}
			alerts.l.Unlock()
		}
	}()
	return &alerts
}

// Put adds alert to cache
func (m *AlertCache) Put(k string, v promApi.Alert) {
	m.l.Lock()
	defer m.l.Unlock()
	it, ok := m.m[k]
	if !ok {
		it = &alert{item: v}
		m.m[k] = it
	}
	it.lastAccessed = time.Now().Unix()
}

// Get gets alert from cache
func (m *AlertCache) Get(k string) *promApi.Alert {
	m.l.Lock()
	defer m.l.Unlock()
	if it, ok := m.m[k]; ok {
		a := it.item
		it.lastAccessed = time.Now().Unix()
		return &a
	}
	return nil
}
