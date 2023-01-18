package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	// PluginName is the name of the Prometheus Botkube plugin.
	PluginName = "prometheus"

	description = "Prometheus plugin polls alerts from configured Prometheus AlertManager."

	pollPeriodInSeconds = 5
)

// Source prometheus source plugin data structure
type Source struct {
	prometheus    *Client
	l             sync.Mutex
	pluginVersion string
	startedAt     time.Time
}

// NewSource returns a new instance of Source.
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
		Description: description,
		JSONSchema:  jsonSchema(),
	}, nil
}

func (p *Source) consumeAlerts(ctx context.Context, config Config, ch chan<- []byte) {
	log := loggerx.New(loggerx.Config{
		Level: config.Log.Level,
	})
	prometheus, err := p.getPrometheusClient(config.URL)
	exitOnError(err, log)
	for {
		alerts, err := prometheus.Alerts(ctx, GetAlertsRequest{
			IgnoreOldAlerts: *config.IgnoreOldAlerts,
			MinAlertTime:    p.startedAt,
			AlertStates:     config.AlertStates,
		})
		log.Errorf("failed to get alerts. %v", err)
		for _, alert := range alerts {
			msg := fmt.Sprintf("[%s][%s][%s] %s", PluginName, alert.Labels["alertname"], alert.State, alert.Annotations["description"])
			ch <- []byte(msg)
		}
		// Fetch alerts periodically with given frequency
		time.Sleep(time.Second * pollPeriodInSeconds)
	}
}

func (p *Source) getPrometheusClient(url string) (*Client, error) {
	p.l.Lock()
	defer p.l.Unlock()
	if p.prometheus == nil {
		c, err := NewClient(url)
		if err != nil {
			return nil, err
		}
		p.prometheus = c
	}
	return p.prometheus, nil
}

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "botkube/prometheus",
			"description": "%s",
			"type": "object",
			"properties": {
				"url": {
					"description": "Prometheus endpoint without api version and resource",
					"type": "string",
					"default": "http://localhost:9090",
				},
				"ignoreOldAlerts": {
					"description": "If set as true, Prometheus source plugin will not send alerts that is created before plugin start time",
					"type": "boolean",
					"enum": ["true", "false"],
					"default": true
				},
				"alertStates": {
					"description": "Only the alerts that have state provided in this config will be sent as notification. https://pkg.go.dev/github.com/prometheus/prometheus/rules#AlertState",
					"type": "array",
					"default": ["firing", "pending", "inactive"]
					"enum: ["firing", "pending", "inactive"]
				},
				"log": {
					"description": "Logging configuration",
					"type": "object",
					"properties": {
						"level": {
							"description": "Log level",
							"type": "string",
							"default": "info",
							"enum: ["info", "debug", "error"]
						}
					}
				},
			},
			"required": []
		}`, description),
	}
}

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
