package keptn

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
	// PluginName is the name of the Keptn Botkube plugin.
	PluginName = "keptn"

	description = "Keptn plugin polls events from configured Keptn API endpoint."

	pollPeriodInSeconds = 5
)

// Source prometheus source plugin data structure
type Source struct {
	pluginVersion string

	source.HandleExternalRequestUnimplemented
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Keptn events
func (p *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	out := source.StreamOutput{Event: make(chan source.Event)}
	config, err := MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	go p.consumeEvents(ctx, config, out.Event)

	return out, nil
}

// Metadata returns metadata of Keptn configuration
func (p *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:          p.pluginVersion,
		Description:      description,
		DocumentationURL: "https://docs.botkube.io/configuration/source/keptn",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
		Recommended: false,
	}, nil
}

func (p *Source) consumeEvents(ctx context.Context, cfg Config, ch chan<- source.Event) {
	keptn, err := NewClient(cfg.URL, cfg.Token)
	log := loggerx.New(cfg.Log)
	exitOnError(err, log)

	ticker := time.NewTicker(time.Second * pollPeriodInSeconds)
	for {
		select {
		case <-ticker.C:
			req := GetEventsRequest{
				Project:  cfg.Project,
				Service:  cfg.Service,
				FromTime: time.Now().Add(-time.Second * pollPeriodInSeconds),
			}
			res, err := keptn.Events(ctx, &req)
			if err != nil {
				log.Errorf("while getting events: %v", err)
			}
			for _, event := range res {
				textFields := []api.TextField{
					{Key: "Source", Value: PluginName},
					{Key: "Type", Value: event.Type},
				}
				if event.Data.Status != "" {
					textFields = append(textFields, api.TextField{Key: "State", Value: event.Data.Status})
				}

				var bulletLists []api.BulletList
				if event.Data.Message != "" {
					bulletLists = []api.BulletList{
						{
							Title: "Description",
							Items: []string{
								event.Data.Message,
							},
						},
					}
				}
				msg := api.Message{
					Type:      api.NonInteractiveSingleSection,
					Timestamp: time.Now(),
					Sections: []api.Section{
						{
							TextFields:  textFields,
							BulletLists: bulletLists,
						},
					},
				}
				ch <- source.Event{
					Message:   msg,
					RawObject: event,
				}
			}
		case <-ctx.Done():
			log.Info("Stopping Keptn event consuming...")
			return
		}
	}
}

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
