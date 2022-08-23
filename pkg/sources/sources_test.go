package sources_test

import (
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/sources"
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

func TestRouter_BuildTable_CreatesRoutesForBoundSources(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	hasRoutes := "apps/v1/deployments"
	hasNoRoutes := "v1/pods"

	router := sources.NewRouter(nil, nil, logger)
	router.AddAnyBindings(config.BotBindings{
		Sources: []string{"k8s-events"},
	})

	cfg := &config.Config{
		Sources: config.IndexableMap[config.Sources]{
			"k8s-events": config.Sources{
				Kubernetes: config.KubernetesSource{
					Resources: []config.Resource{
						{
							Name: hasRoutes,
							Namespaces: config.Namespaces{
								Include: []string{"default"},
							},
							Events: []config.EventType{
								config.CreateEvent,
								config.DeleteEvent,
								config.UpdateEvent,
								config.ErrorEvent,
							},
							UpdateSetting: config.UpdateSetting{
								Fields:      []string{"status.availableReplicas"},
								IncludeDiff: true,
							},
						},
					},
				},
			},
			"k8s-ignored": config.Sources{
				Kubernetes: config.KubernetesSource{
					Resources: []config.Resource{
						{
							Name: hasNoRoutes,
							Namespaces: config.Namespaces{
								Include: []string{"all"},
							},
							Events: []config.EventType{
								config.ErrorEvent,
							},
							UpdateSetting: config.UpdateSetting{
								Fields:      []string{""},
								IncludeDiff: false,
							},
						},
					},
				},
			},
		},
	}

	router = router.BuildTable(cfg)
	assert.Len(t, router.GetSourceRoutes(hasRoutes, config.CreateEvent), 1)
	assert.Len(t, router.GetSourceRoutes(hasRoutes, config.UpdateEvent), 1)
	assert.Len(t, router.GetSourceRoutes(hasRoutes, config.DeleteEvent), 1)
	assert.Len(t, router.GetSourceRoutes(hasRoutes, config.ErrorEvent), 1)
	assert.Len(t, router.GetSourceRoutes(hasNoRoutes, config.ErrorEvent), 0)
}
