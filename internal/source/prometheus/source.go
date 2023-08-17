package prometheus

import (
	"context"
	"fmt"
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

	description = "Get notifications about alerts polled from configured Prometheus AlertManager."

	pollPeriodInSeconds = 5
)

// Source prometheus source plugin data structure
type Source struct {
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
	out := source.StreamOutput{Event: make(chan source.Event)}
	config, err := MergeConfigs(input.Config)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	go p.consumeAlerts(ctx, config, out.Event)

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

func (p *Source) consumeAlerts(ctx context.Context, cfg Config, ch chan<- source.Event) {
	log := loggerx.New(cfg.Log)
	prometheus, err := NewClient(cfg.URL)
	exitOnError(err, log)

	for {
		alerts, err := prometheus.Alerts(ctx, GetAlertsRequest{
			IgnoreOldAlerts: *cfg.IgnoreOldAlerts,
			MinAlertTime:    p.startedAt,
			AlertStates:     cfg.AlertStates,
		})
		if err != nil {
			log.Errorf("failed to get alerts. %v", err)
		}
		for _, alert := range alerts {
			msg := api.Message{
				Type:      api.NonInteractiveSingleSection,
				Timestamp: time.Now(),
				Sections: []api.Section{
					{
						TextFields: []api.TextField{
							{Key: "Source", Value: PluginName},
							{Key: "Alert Name", Value: string(alert.Labels["alertname"])},
							{Key: "State", Value: string(alert.State)},
						},
						BulletLists: []api.BulletList{
							{
								Title: "Description",
								Items: []string{
									string(alert.Annotations["description"]),
								},
							},
						},
					},
				},
			}
			ch <- source.Event{
				Message:   msg,
				RawObject: alert,
			}
		}
		// Fetch alerts periodically with given frequency
		time.Sleep(time.Second * pollPeriodInSeconds)
	}
}

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
		  "$schema": "http://json-schema.org/draft-07/schema#",
		  "title": "Prometheus",
		  "description": "%s",
		  "type": "object",
		  "properties": {
			"url": {
			  "title": "Endpoint",
			  "description": "Prometheus endpoint without API version and resource.",
			  "type": "string",
			  "format": "uri"
			},
			"ignoreOldAlerts": {
			  "title": "Ignore old alerts",
			  "description": "If set to true, Prometheus source plugin will not send alerts that is created before the plugin start time.",
			  "type": "boolean",
			  "default": true
			},
			"alertStates": {
			  "title": "Alert states",
			  "description": "Only the alerts that have state provided in this config will be sent as notification. https://pkg.go.dev/github.com/prometheus/prometheus/rules#AlertState",
			  "type": "array",
			  "default": [
				"firing",
				"pending",
				"inactive"
			  ],
			  "items": {
				"type": "string",
				"title": "Alert state",
				"oneOf": [
				  {
					"const": "firing",
					"title": "Firing"
				  },
				  {
					"const": "pending",
					"title": "Pending"
				  },
				  {
					"const": "inactive",
					"title": "Inactive"
				  }
				]
			  },
			  "uniqueItems": true,
			  "minItems": 1
			},
			"log": {
			  "title": "Logging",
			  "description": "Logging configuration for the plugin.",
			  "type": "object",
			  "properties": {
				"level": {
				  "title": "Log Level",
				  "description": "Define log level for the plugin. Ensure that Botkube has plugin logging enabled for standard output.",
				  "type": "string",
				  "default": "info",
				  "oneOf": [
					{
					  "const": "panic",
					  "title": "Panic"
					},
					{
					  "const": "fatal",
					  "title": "Fatal"
					},
					{
					  "const": "error",
					  "title": "Error"
					},
					{
					  "const": "warn",
					  "title": "Warning"
					},
					{
					  "const": "info",
					  "title": "Info"
					},
					{
					  "const": "debug",
					  "title": "Debug"
					},
					{
					  "const": "trace",
					  "title": "Trace"
					}
				  ]
				},
				"disableColors": {
				  "type": "boolean",
				  "default": false,
				  "description": "If enabled, disables color logging output.",
				  "title": "Disable Colors"
				}
			  }
			}
		  },
		  "required": ["url"]
		}`, description),
	}
}

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
