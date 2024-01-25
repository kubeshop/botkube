package kubernetes

import (
	"fmt"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/golden"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
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
	for _, testCase := range tests {
		tc := testCase
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

func TestRouterListMergingNestedFields(t *testing.T) {
	// given
	router := NewRouter(nil, nil, loggerx.NewNoop())

	var cfg config.Config
	fixConfig, err := os.ReadFile(filepath.Join("testdata", t.Name(), "override-fields-config.yaml"))
	require.NoError(t, err)

	err = yaml.Unmarshal(fixConfig, &cfg)
	require.NoError(t, err)

	// when
	router = router.BuildTable(&cfg)

	// then
	for key := range router.table {
		out, err := yaml.Marshal(router.table[key])
		require.NoError(t, err)
		filename := fmt.Sprintf("route-%s.golden.yaml", strings.ReplaceAll(key, "/", "."))
		golden.Assert(t, string(out), filepath.Join(t.Name(), filename))
	}
}
