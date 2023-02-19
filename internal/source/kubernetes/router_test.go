package kubernetes

import (
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/loggerx"
)

func TestRouter_BuildTable_CreatesRoutesWithProperEventsList(t *testing.T) {
	const hasRoutes = "apps/v1/deployments"

	tests := []struct {
		name     string
		givenCfg config.Config
	}{
		{
			name: "Events defined on top-level but override by resource once",
			givenCfg: config.Config{

				Event: &config.KubernetesEvent{
					Types: []config.EventType{
						config.CreateEvent,
						config.ErrorEvent,
					},
				},
				Resources: []config.Resource{
					{
						Type: hasRoutes,
						Namespaces: config.RegexConstraints{
							Include: []string{"default"},
						},
						Event: config.KubernetesEvent{
							Types: []config.EventType{
								config.CreateEvent,
								config.DeleteEvent,
								config.UpdateEvent,
								config.ErrorEvent,
							},
						},
						UpdateSetting: config.UpdateSetting{
							Fields:      []string{"status.availableReplicas"},
							IncludeDiff: true,
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter(nil, nil, loggerx.NewNoop())

			router = router.BuildTable(&tc.givenCfg)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.CreateEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.UpdateEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.DeleteEvent), 1)
			assert.Len(t, router.getSourceRoutes(hasRoutes, config.ErrorEvent), 1)
		})
	}
}
