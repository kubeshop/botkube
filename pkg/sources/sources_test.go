package sources_test

import (
	"testing"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_GetBoundSources_UsesAddedBindings(t *testing.T) {
	router := sources.NewRouter(nil, nil, nil)

	router.AddAnyBindings(config.BotBindings{
		Sources: []string{"k8s-events"},
	})
	router.AddAnyBindingsByName(config.IdentifiableMap[config.ChannelBindingsByName]{
		"this": config.ChannelBindingsByName{
			Name: "channel-name",
			Bindings: config.BotBindings{
				Sources: []string{"k8s-events", "k8s-other"},
			},
		},
	})
	router.AddAnyBindingsByID(config.IdentifiableMap[config.ChannelBindingsByID]{
		"that": config.ChannelBindingsByID{
			ID: "channel-id",
			Bindings: config.BotBindings{
				Sources: []string{"k8s-events"},
			},
		},
	})

	candidates := config.IndexableMap[config.Sources]{
		"k8s-events": config.Sources{
			Kubernetes: config.KubernetesSource{},
		},
		"k8s-other": config.Sources{
			Kubernetes: config.KubernetesSource{},
		},
		"k8s-ignored": config.Sources{
			Kubernetes: config.KubernetesSource{},
		},
	}

	boundSources := router.GetBoundSources(candidates)

	require.Len(t, boundSources, 2)
	assert.NotContains(t, boundSources, "k8s-ignored")
}
