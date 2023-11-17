package batched

import "github.com/kubeshop/botkube/pkg/config"

type HeartbeatProperties struct {
	TimeWindowInHours int                         `json:"timeWindowInHours"`
	EventsCount       int                         `json:"eventsCount"`
	Sources           map[string]SourceProperties `json:"sources"`
}

type SourceProperties struct {
	EventsCount int           `json:"eventsCount"`
	Events      []SourceEvent `json:"events"`
}

type SourceEvent struct {
	IntegrationType       config.IntegrationType         `json:"integrationType"`
	Platform              config.CommPlatformIntegration `json:"platform"`
	PluginName            string                         `json:"pluginName"`
	AnonymizedEventFields map[string]any                 `json:"anonymizedEventFields"`
	Success               bool                           `json:"success"`
	Error                 *string                        `json:"error"`
}
