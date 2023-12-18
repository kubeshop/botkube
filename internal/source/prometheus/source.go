package prometheus

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
)

var _ source.Source = (*Source)(nil)

var (
	//go:embed config_schema.json
	configJSONSchema string
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

	source.HandleExternalRequestUnimplemented
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
	config, err := MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	go p.consumeAlerts(ctx, config, out.Event)

	return out, nil
}

// Metadata returns metadata of prometheus configuration
func (p *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:          p.pluginVersion,
		Description:      description,
		DocumentationURL: "https://docs.botkube.io/configuration/source/prometheus",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
		Recommended: false,
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

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
