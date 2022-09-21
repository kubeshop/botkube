package sources

import (
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestRouter_GetBoundSources_UsesAddedBindings(t *testing.T) {
	router := NewRouter(nil, nil, nil)

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

	candidates := map[string]config.Sources{
		"k8s-events": {
			Kubernetes: config.KubernetesSource{},
		},
		"k8s-other": {
			Kubernetes: config.KubernetesSource{},
		},
		"k8s-ignored": {
			Kubernetes: config.KubernetesSource{},
		},
	}

	boundSources := router.GetBoundSources(candidates)

	require.Len(t, boundSources, 2)
	assert.NotContains(t, boundSources, "k8s-ignored")
}

func TestRouter_BuildTable_CreatesRoutesWithProperEventsList(t *testing.T) {
	const hasRoutes = "apps/v1/deployments"
	logger, _ := logtest.NewNullLogger()

	tests := []struct {
		name     string
		givenCfg config.Config
	}{
		{
			name: "Events defined on resource level",
			givenCfg: config.Config{
				Sources: map[string]config.Sources{
					"k8s-events": {
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
				},
			},
		},
		{
			name: "Events defined on top-level",
			givenCfg: config.Config{
				Sources: map[string]config.Sources{
					"k8s-events": {
						Kubernetes: config.KubernetesSource{
							Events: []config.EventType{
								config.CreateEvent,
								config.DeleteEvent,
								config.UpdateEvent,
								config.ErrorEvent,
							},
							Resources: []config.Resource{
								{
									Name: hasRoutes,
									Namespaces: config.Namespaces{
										Include: []string{"default"},
									},
									UpdateSetting: config.UpdateSetting{
										Fields:      []string{"status.availableReplicas"},
										IncludeDiff: true,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Events defined on top-level but override by resource once",
			givenCfg: config.Config{
				Sources: map[string]config.Sources{
					"k8s-events": {
						Kubernetes: config.KubernetesSource{
							Events: []config.EventType{
								config.CreateEvent,
								config.ErrorEvent,
							},
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
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(nil, nil, logger)
			router.AddAnyBindings(config.BotBindings{
				Sources: []string{"k8s-events"},
			})

			router = router.BuildTable(&tc.givenCfg)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.CreateEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.UpdateEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.DeleteEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.ErrorEvent), 1)
		})
	}
}

func TestRouter_BuildTable_CreatesRoutesForBoundSources(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	hasRoutes := "apps/v1/deployments"
	hasNoRoutes := "v1/pods"

	router := NewRouter(nil, nil, logger)
	router.AddAnyBindings(config.BotBindings{
		Sources: []string{"k8s-events"},
	})

	cfg := &config.Config{
		Sources: map[string]config.Sources{
			"k8s-events": {
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
			"k8s-ignored": {
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
	assert.Len(t, router.getSourceRoutes(hasRoutes, config.CreateEvent), 1)
	assert.Len(t, router.getSourceRoutes(hasRoutes, config.UpdateEvent), 1)
	assert.Len(t, router.getSourceRoutes(hasRoutes, config.DeleteEvent), 1)
	assert.Len(t, router.getSourceRoutes(hasRoutes, config.ErrorEvent), 1)
	assert.Len(t, router.getSourceRoutes(hasNoRoutes, config.ErrorEvent), 0)
}

func TestRouter_BuildTable_CreatesRoutesWithNamespacesPresetFromKubernetesSource(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	testCases := []struct {
		Name     string
		Input    *config.Config
		Expected config.Namespaces
	}{
		{
			Name: "Use sources Namespaces",
			Input: &config.Config{
				Sources: map[string]config.Sources{
					"k8s-events": {
						Kubernetes: config.KubernetesSource{
							Namespaces: config.Namespaces{
								Include: []string{"botkube"},
								Exclude: []string{"default"},
							},
							Resources: []config.Resource{
								{
									Name: "apps/v1/deployments",
									Events: []config.EventType{
										config.CreateEvent,
									},
								},
							},
						},
					},
				},
			},
			Expected: config.Namespaces{
				Include: []string{"botkube"},
				Exclude: []string{"default"},
			},
		},
		{
			Name: "Override sources Namespaces",
			Input: &config.Config{
				Sources: map[string]config.Sources{
					"k8s-events": {
						Kubernetes: config.KubernetesSource{
							Namespaces: config.Namespaces{
								Include: []string{"botkube"},
								Exclude: []string{"default"},
							},
							Resources: []config.Resource{
								{
									Name: "apps/v1/deployments",
									Namespaces: config.Namespaces{
										Include: []string{".*"},
									},
									Events: []config.EventType{
										config.CreateEvent,
									},
								},
							},
						},
					},
				},
			},
			Expected: config.Namespaces{
				Include: []string{".*"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			routes := NewRouter(nil, nil, logger).
				AddAnyBindings(config.BotBindings{Sources: []string{"k8s-events"}}).
				BuildTable(tc.Input).
				getSourceRoutes("apps/v1/deployments", config.CreateEvent)

			assert.Len(t, routes, 1)
			assert.Equal(t, tc.Expected, routes[0].namespaces)
		})
	}
}

func TestSetEventRouteForRecommendationsIfShould(t *testing.T) {
	// given
	resForRecomms := map[string]config.EventType{
		"v1/pods":                      config.CreateEvent,
		"networking.k8s.io/v1/ingress": config.CreateEvent,
	}
	resourceName := "v1/pods"
	srcGroupName := "foo"

	testCases := []struct {
		Name     string
		Input    map[config.EventType][]route
		Expected map[config.EventType][]route
	}{
		{
			Name: "Append",
			Input: map[config.EventType][]route{
				config.CreateEvent: {},
				config.UpdateEvent: {{source: "foo"}, {source: "bar"}},
			},
			Expected: map[config.EventType][]route{
				config.CreateEvent: {{source: "foo", namespaces: config.Namespaces{Include: []string{config.AllNamespaceIndicator}}}},
				config.UpdateEvent: {{source: "foo"}, {source: "bar"}},
			},
		},
		{
			Name: "Override",
			Input: map[config.EventType][]route{
				config.CreateEvent: {
					{
						source: "bar",
					},
					{
						source: "foo",
						namespaces: config.Namespaces{
							Include: []string{"foo", "bar"},
							Exclude: []string{"default"},
						},
					},
					{
						source: "baz",
					},
				},
				config.UpdateEvent: {{source: "foo"}, {source: "bar"}},
			},
			Expected: map[config.EventType][]route{
				config.CreateEvent: {
					{
						source: "bar",
					},
					{
						source: "foo",
						namespaces: config.Namespaces{
							Include: []string{config.AllNamespaceIndicator},
						},
					},
					{
						source: "baz",
					},
				},
				config.UpdateEvent: {{source: "foo"}, {source: "bar"}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			r := &Router{}

			// when
			r.setEventRouteForRecommendationsIfShould(&tc.Input, resForRecomms, srcGroupName, resourceName)

			// then
			assert.Equal(t, tc.Expected, tc.Input)
		})
	}
}
