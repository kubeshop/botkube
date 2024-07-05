package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/httpx"
	"github.com/kubeshop/botkube/pkg/plugin"
)

const (
	healthEndpointName = "/healthz"
)

// Notifier represents notifier interface
type Notifier interface {
	GetStatus() PlatformStatus
}

// Checker gives health bot agent status.
type Checker struct {
	applicationStarted bool
	ctx                context.Context
	config             *config.Config
	pluginHealthStats  *plugin.HealthStats
	notifiers          map[string]Notifier
}

// NewChecker create new health checker.
func NewChecker(ctx context.Context, config *config.Config, stats *plugin.HealthStats) Checker {
	return Checker{
		applicationStarted: false,
		ctx:                ctx,
		config:             config,
		pluginHealthStats:  stats,
	}
}

// MarkAsReady marks bot as ready
func (h *Checker) MarkAsReady() {
	h.applicationStarted = true
}

// IsReady gets info if bot is ready
func (h *Checker) IsReady() bool {
	return h.applicationStarted
}

// ServeHTTP serves status on health endpoint.
func (h *Checker) ServeHTTP(resp http.ResponseWriter, _ *http.Request) {
	statusCode := http.StatusOK
	if !h.IsReady() {
		statusCode = http.StatusServiceUnavailable
	}
	resp.Header().Set("Content-Type", "application/json")

	status := h.GetStatus()
	respJSon, err := json.Marshal(status)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(statusCode)
	_, _ = fmt.Fprint(resp, string(respJSon))
}

// NewServer creates http server for health checker.
func (h *Checker) NewServer(log logrus.FieldLogger, port string) *httpx.Server {
	addr := fmt.Sprintf(":%s", port)
	router := mux.NewRouter()
	router.Handle(healthEndpointName, h)
	return httpx.NewServer(log, addr, router)
}

// SetNotifiers sets platform bots instances.
func (h *Checker) SetNotifiers(notifiers map[string]Notifier) {
	h.notifiers = notifiers
}

func (h *Checker) GetStatus() *Status {
	pluginsStats := make(map[string]PluginStatus)
	h.collectSourcePluginsStatuses(pluginsStats)
	h.collectExecutorPluginsStatuses(pluginsStats)

	return &Status{
		Botkube: BotStatus{
			Status: h.getBotkubeStatus(),
		},
		Plugins:   pluginsStats,
		Platforms: h.getPlatformsStatus(),
	}
}

func (h *Checker) collectSourcePluginsStatuses(plugins map[string]PluginStatus) {
	if h.config == nil {
		return
	}
	for pluginConfigName, sourceValues := range h.config.Sources {
		for pluginName, pluginValues := range sourceValues.GetPlugins() {
			h.collectPluginStatus(plugins, pluginConfigName, pluginName, pluginValues.Enabled)
		}
	}
}

func (h *Checker) collectExecutorPluginsStatuses(plugins map[string]PluginStatus) {
	if h.config == nil {
		return
	}
	for pluginConfigName, execValues := range h.config.Executors {
		for pluginName, pluginValues := range execValues.GetPlugins() {
			h.collectPluginStatus(plugins, pluginConfigName, pluginName, pluginValues.Enabled)
		}
	}
}

func (h *Checker) collectPluginStatus(plugins map[string]PluginStatus, pluginConfigName string, pluginName string, enabled bool) {
	status, restarts, threshold, _ := h.pluginHealthStats.GetStats(pluginName)
	plugins[pluginConfigName] = PluginStatus{
		Enabled:  enabled,
		Status:   status,
		Restarts: fmt.Sprintf("%d/%d", restarts, threshold),
	}
}

func (h *Checker) getBotkubeStatus() BotkubeStatus {
	if h.applicationStarted {
		return BotkubeStatusHealthy
	}
	return BotkubeStatusUnhealthy
}

func (h *Checker) getPlatformsStatus() platformStatuses {
	defaultStatuses := platformStatuses{}
	if h.notifiers != nil {
		for key, notifier := range h.notifiers {
			defaultStatuses[key] = notifier.GetStatus()
		}
	}

	return defaultStatuses
}
