package plugin

import (
	"sync"
	"time"
)

const (
	pluginRunning     = "Running"
	pluginDeactivated = "Deactivated"
)

// HealthStats holds information about plugin health and restarts.
type HealthStats struct {
	sync.RWMutex
	pluginStats            map[string]pluginStats
	globalRestartThreshold int
}

type pluginStats struct {
	restartCount       int
	restartThreshold   int
	lastTransitionTime string
}

// NewHealthStats returns a new HealthStats instance.
func NewHealthStats(threshold int) *HealthStats {
	return &HealthStats{
		pluginStats:            map[string]pluginStats{},
		globalRestartThreshold: threshold,
	}
}

// Increment increments restart count for a plugin.
func (h *HealthStats) Increment(plugin string) {
	h.Lock()
	defer h.Unlock()
	if _, ok := h.pluginStats[plugin]; !ok {
		h.pluginStats[plugin] = pluginStats{}
	}
	count := h.pluginStats[plugin].restartCount + 1
	if count > h.globalRestartThreshold {
		count = h.globalRestartThreshold
	}
	h.pluginStats[plugin] = pluginStats{
		restartCount:       count,
		lastTransitionTime: time.Now().Format(time.DateTime),
		restartThreshold:   h.globalRestartThreshold,
	}
}

// GetRestartCount returns restart count for a plugin.
func (h *HealthStats) GetRestartCount(plugin string) int {
	h.RLock()
	defer h.RUnlock()
	if _, ok := h.pluginStats[plugin]; !ok {
		return 0
	}
	return h.pluginStats[plugin].restartCount
}

// GetStats returns plugin status, restart count, restart threshold and last transition time.
func (h *HealthStats) GetStats(plugin string) (status string, restarts int, threshold int, timestamp string) {
	h.RLock()
	defer h.RUnlock()
	if _, ok := h.pluginStats[plugin]; !ok {
		status = pluginRunning
		threshold = h.globalRestartThreshold
		return
	}

	status = pluginRunning
	if h.pluginStats[plugin].restartCount >= h.pluginStats[plugin].restartThreshold {
		status = pluginDeactivated
	}
	restarts = h.pluginStats[plugin].restartCount
	threshold = h.pluginStats[plugin].restartThreshold
	timestamp = h.pluginStats[plugin].lastTransitionTime
	return
}
