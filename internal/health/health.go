package health

import (
	"context"
	"encoding/json"
	"github.com/kubeshop/botkube/pkg/bot"

	"fmt"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/kubeshop/botkube/internal/httpx"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/sirupsen/logrus"
)

type BotkubeStatus string

const (
	BotkubeStatusHealthy   BotkubeStatus = "Healthy"
	BotkubeStatusUnhealthy BotkubeStatus = "Unhealthy"
)

const (
	healthEndpointName = "/healthz"
)

type Checker struct {
	applicationStarted bool
	ctx                context.Context
	config             *config.Config
	pluginHealthStats  *plugin.HealthStats
	bots               map[string]bot.Bot
}

type pluginStatuses struct {
	Enabled  bool
	Status   string
	Restarts string
}

type botStatus struct {
	Status BotkubeStatus
}

type platformStatuses map[string]bot.Status

type Status struct {
	Botkube   botStatus
	Plugins   map[string]pluginStatuses
	Platforms platformStatuses
}

func NewChecker(ctx context.Context, config *config.Config, stats *plugin.HealthStats) Checker {
	return Checker{
		applicationStarted: false,
		ctx:                ctx,
		config:             config,
		pluginHealthStats:  stats,
	}
}

func (h *Checker) MarkAsReady() {
	h.applicationStarted = true
}

func (h *Checker) IsReady() bool {
	return h.applicationStarted
}

func (h *Checker) GetStatus() (*Status, error) {
	pluginsStats := make(map[string]pluginStatuses)
	h.getSourcePluginsStatuses(pluginsStats)
	h.getExecutorPluginsStatuses(pluginsStats)
	platformStatuses, err := h.getPlatformsStatus()
	if err != nil {
		return nil, err
	}

	return &Status{
		Botkube: botStatus{
			Status: h.getBotkubeStatus(),
		},
		Plugins:   pluginsStats,
		Platforms: platformStatuses,
	}, nil
}

func (h *Checker) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if h.IsReady() {
		resp.Header().Set("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
		status, err := h.GetStatus()
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(resp, "Internal Error")
		}
		respJSon, err := json.Marshal(status)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(resp, "Internal Error")
		}
		_, _ = fmt.Fprint(resp, string(respJSon))
	} else {
		resp.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(resp, "unavailable")
	}
}

func (h *Checker) NewServer(log logrus.FieldLogger, port string) *httpx.Server {
	addr := fmt.Sprintf(":%s", port)
	router := mux.NewRouter()
	router.Handle(healthEndpointName, h)
	return httpx.NewServer(log, addr, router)
}

func (h *Checker) SetBots(bots map[string]bot.Bot) {
	h.bots = bots
}

func (h *Checker) getSourcePluginsStatuses(plugins map[string]pluginStatuses) {
	for _, sourceValues := range h.config.Sources {
		for pluginName, pluginValues := range sourceValues.GetPlugins() {
			h.getPluginStatus(plugins, pluginName, pluginValues.Enabled)
		}
	}
}

func (h *Checker) getExecutorPluginsStatuses(plugins map[string]pluginStatuses) {
	for _, execValues := range h.config.Executors {
		for pluginName, pluginValues := range execValues.GetPlugins() {
			h.getPluginStatus(plugins, pluginName, pluginValues.Enabled)
		}
	}
}

func (h *Checker) getPluginStatus(plugins map[string]pluginStatuses, pluginName string, enabled bool) {
	status, restarts, threshold, _ := h.pluginHealthStats.GetStats(pluginName)
	plugins[pluginName] = pluginStatuses{
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

func (h *Checker) getPlatformsStatus() (platformStatuses, error) {
	defaultStatuses := platformStatuses{}
	if h.bots != nil {
		for key, botInstance := range h.bots {
			defaultStatuses[key] = botInstance.GetStatus()
		}
	}

	return defaultStatuses, nil
}
