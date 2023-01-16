package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"gopkg.in/yaml.v3"
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
}

func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams prometheus alerts
func (p *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	out := source.StreamOutput{Output: make(chan []byte)}
	config, err := mergeConfigs(input.Configs)
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
		alerts, _ := prometheus.Alerts(ctx)
		for _, alert := range alerts {
			ch <- []byte(fmt.Sprintf("%+v", alert))
		}
		time.Sleep(time.Second * 5)
	}
}

func mergeConfigs(configs []*source.Config) (Config, error) {
	finalCfg := Config{}
	for _, inputCfg := range configs {
		var cfg Config
		err := yaml.Unmarshal(inputCfg.RawYAML, &cfg)
		if err != nil {
			return Config{}, err
		}
		if cfg.URL == "" {
			continue
		}
		finalCfg.URL = cfg.URL
	}
	return finalCfg, nil
}

func (p *Source) getPrometheusClient(url string) *Client {
	p.l.Lock()
	defer p.l.Unlock()
	if p.prometheus == nil {
		return NewClient(url)
	}
	return p.prometheus
}
