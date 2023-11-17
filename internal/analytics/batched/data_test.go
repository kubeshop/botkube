package batched

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestData(t *testing.T) {
	// given
	event := SourceEvent{
		IntegrationType: config.BotIntegrationType,
		Platform:        config.DiscordCommPlatformIntegration,
		PluginName:      "botkube/kubernetes",
		AnonymizedEventFields: map[string]any{
			"foo": "bar",
		},
		Success: true,
	}
	event2 := SourceEvent{
		IntegrationType: config.SinkIntegrationType,
		Platform:        config.ElasticsearchCommPlatformIntegration,
		PluginName:      "botkube/kubernetes",
		AnonymizedEventFields: map[string]any{
			"els": true,
		},
		Success: true,
	}
	event3 := SourceEvent{
		IntegrationType:       config.BotIntegrationType,
		Platform:              config.CloudSlackCommPlatformIntegration,
		PluginName:            "botkube/argocd",
		AnonymizedEventFields: nil,
		Success:               true,
	}

	// when
	data := NewData(1)
	// then
	assert.Equal(t, 1, data.heartbeatProperties.TimeWindowInHours)

	// when
	data.AddSourceEvent(event)
	data.AddSourceEvent(event2)
	// then
	expected := HeartbeatProperties{
		TimeWindowInHours: 1,
		Sources: map[string]SourceProperties{
			"botkube/kubernetes": {
				EventsCount: 2,
				Events: []SourceEvent{
					event,
					event2,
				},
			},
		},
		EventsCount: 2,
	}
	assert.Equal(t, expected, data.heartbeatProperties)

	// when
	data.IncrementTimeWindowInHours()
	// then
	assert.Equal(t, 2, data.heartbeatProperties.TimeWindowInHours)

	// when
	data.AddSourceEvent(event3)
	// then
	expected = HeartbeatProperties{
		TimeWindowInHours: 2,
		Sources: map[string]SourceProperties{
			"botkube/kubernetes": {
				EventsCount: 2,
				Events: []SourceEvent{
					event,
					event2,
				},
			},
			"botkube/argocd": {
				EventsCount: 1,
				Events: []SourceEvent{
					event3,
				},
			},
		},
		EventsCount: 3,
	}
	assert.Equal(t, expected, data.heartbeatProperties)

	// when
	data.Reset()
	// then
	assert.Equal(t, 1, data.heartbeatProperties.TimeWindowInHours)
	assert.Equal(t, 0, data.heartbeatProperties.EventsCount)
	assert.Len(t, data.heartbeatProperties.Sources, 0)
}

func TestData_EventDetailsLimit(t *testing.T) {
	// given
	data := NewData(1)
	addEvent1Count := 50
	addEvent2Count := 70
	addEvent3Count := 30

	totalCount := addEvent1Count + addEvent2Count + addEvent3Count
	expectedKubernetesEventCount := addEvent1Count + addEvent3Count
	expectedKubernetesEventDetailsLen := addEvent1Count
	expectedArgoCDEventCount := addEvent2Count

	kubernetesPlugin := "botkube/kubernetes"
	argoCDPlugin := "botkube/argocd"

	// when
	for i := 0; i < addEvent1Count; i++ {
		data.AddSourceEvent(SourceEvent{
			IntegrationType: config.BotIntegrationType,
			Platform:        config.DiscordCommPlatformIntegration,
			PluginName:      kubernetesPlugin,
			AnonymizedEventFields: map[string]any{
				"foo": "bar",
			},
			Success: true,
		})
	}

	for i := 0; i < addEvent2Count; i++ {
		data.AddSourceEvent(SourceEvent{
			IntegrationType:       config.BotIntegrationType,
			Platform:              config.CloudSlackCommPlatformIntegration,
			PluginName:            argoCDPlugin,
			AnonymizedEventFields: nil,
			Success:               true,
		})
	}

	for i := 0; i < addEvent3Count; i++ {
		data.AddSourceEvent(SourceEvent{
			IntegrationType: config.SinkIntegrationType,
			Platform:        config.ElasticsearchCommPlatformIntegration,
			PluginName:      kubernetesPlugin,
			AnonymizedEventFields: map[string]any{
				"foo": "bar",
			},
			Success: true,
		})
	}

	// then
	assert.Equal(t, totalCount, data.heartbeatProperties.EventsCount)
	assert.Len(t, data.heartbeatProperties.Sources, 2)

	assert.Equal(t, expectedKubernetesEventCount, data.heartbeatProperties.Sources[kubernetesPlugin].EventsCount)
	assert.Len(t, data.heartbeatProperties.Sources[kubernetesPlugin].Events, expectedKubernetesEventDetailsLen)

	assert.Equal(t, expectedArgoCDEventCount, data.heartbeatProperties.Sources[argoCDPlugin].EventsCount)
	assert.Len(t, data.heartbeatProperties.Sources[argoCDPlugin].Events, maxEventDetailsCount-addEvent1Count)
}
