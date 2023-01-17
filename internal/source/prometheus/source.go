package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	// PluginName is the name of the Prometheus Botkube plugin.
	PluginName = "prometheus"
)

// Source prometheus source plugin data structure
type Source struct {
	prometheus    *Client
	l             sync.Mutex
	pluginVersion string
	startedAt     time.Time
}

func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
		startedAt:     time.Now(),
	}
}

// Stream streams prometheus alerts
func (p *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	out := source.StreamOutput{Output: make(chan []byte)}
	config, err := MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	go p.consumeAlerts(ctx, config, out.Output)
	return out, nil
}

// Metadata returns metadata of prometheus configuration
func (p *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     p.pluginVersion,
		Description: "Prometheus polls alerts from Prometheus alert manager to report in Botkube",
	}, nil
}

func (p *Source) consumeAlerts(ctx context.Context, config Config, ch chan<- []byte) {
	prometheus := p.getPrometheusClient(config.URL)
	for {
		alerts, _ := prometheus.Alerts(ctx, GetAlertsRequest{
			IgnoreOldAlerts: *config.IgnoreOldAlerts,
			MinAlertTime:    p.startedAt,
			AlertStates:     config.AlertStates,
		})
		for _, alert := range alerts {
			msg := fmt.Sprintf("[%s][%s][%s] %s", PluginName, alert.Labels["alertname"], alert.State, alert.Annotations["description"])
			ch <- []byte(msg)
		}
		time.Sleep(time.Second * 5)
	}
}

func (p *Source) getPrometheusClient(url string) *Client {
	p.l.Lock()
	defer p.l.Unlock()
	if p.prometheus == nil {
		return NewClient(url)
	}
	return p.prometheus
}
