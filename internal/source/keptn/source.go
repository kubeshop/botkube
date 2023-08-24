package keptn

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

var _ source.Source = (*Source)(nil)

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
		Version:     p.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(),
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

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"title": "Keptn",
			"description": "%s",
			"type": "object",
			"properties": {
			  "url": {
				"description": "Keptn API Gateway URL",
				"type": "string",
				"default": "http://api-gateway-nginx.keptn.svc.cluster.local/api",
				"title": "Endpoint URL"
			  },
			  "token": {
				"description": "Keptn API Token to access events through API Gateway",
				"type": "string",
				"title": "Keptn API Token"
			  },
			  "project": {
				"description": "Keptn Project",
				"type": "string",
				"title": "Project"
			  },
			  "service": {
				"description": "Keptn Service name under the project",
				"type": "string",
				"title": "Service"
			  }
			},
            "required": [
              "token"
            ]
		  }`, description),
	}
}

func exitOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
