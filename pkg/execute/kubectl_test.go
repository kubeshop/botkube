package execute

import (
	"strings"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestKubectlExecute(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	tests := []struct {
		name string

		command              string
		channelNotAuthorized bool
		resMapping           ResourceMapping
		kubectlCfg           config.Kubectl
		expKubectlExecuted   bool
		expOutMsg            string
	}{
		{
			name: "Should forbid execution from not authorized channel when restrictions are enabled",

			command:              "get pod --cluster-name test",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled:        true,
				RestrictAccess: true,
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, this channel is not authorized to execute kubectl command on cluster 'test'.",
		},
		{
			name: "Should omit message if channel is not authorized but we are not the target cluster",

			command:              "get pod --cluster-name other-cluster",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled:        true,
				RestrictAccess: true,
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "",
		},
		{
			name: "Should omit message if channel is not authorized and there is no --cluster-name flag",

			command:              "get pod",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled:        true,
				RestrictAccess: true,
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "",
		},
		{
			name: "Should forbid execution if kubectl is disabled in config",

			command:    "get pod --cluster-name test",
			kubectlCfg: config.Kubectl{Enabled: false},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster 'test'.",
		},
		{
			name: "Should forbid execution if resource is not allowed in config",

			command:    "get pod",
			kubectlCfg: config.Kubectl{Enabled: true},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command is not authorized to work with 'pod' resources on cluster 'test'.",
		},
		{
			name: "Should forbid execution if namespace is not allowed in config",

			command:    "get pod",
			kubectlCfg: config.Kubectl{Enabled: true},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command cannot be executed in the 'default' Namespace on cluster 'test'.",
		},
		{
			name: "Should use default Namespace from config if not specified in command",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled:          true,
				DefaultNamespace: "from-config",
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command cannot be executed in the 'from-config' Namespace on cluster 'test'.",
		},
		{
			name: "Should explicitly use 'default' Namespace if not specified both in command and config",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command cannot be executed in the 'default' Namespace on cluster 'test'.",
		},
		{
			name: "Should forbid execution in not allowed namespace",

			command: "get pod -n team-b",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command cannot be executed in the 'team-b' Namespace on cluster 'test'.",
		},
		{
			name: "Should forbid execution if all namespace are allowed but command namespace is explicitly ignored in config",

			command: "get pod -n team-b",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{config.AllNamespaceIndicator},
					Ignore:  []string{"team-b"},
				},
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: false,
			expOutMsg:          "Sorry, the kubectl command cannot be executed in the 'team-b' Namespace on cluster 'test'.",
		},
		{
			name: "Should allow execution if verb, resource, and all namespaces are allowed",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{config.AllNamespaceIndicator},
				},
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: true,
			expOutMsg:          "Cluster: test\nkubectl executed",
		},
		{
			name: "Should allow execution if verb, resource, and a given namespace are allowed",

			command: "get pod -n team-a",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: true,
			expOutMsg:          "Cluster: test\nkubectl executed",
		},
		{
			name: "Should allow execution from not authorized channel if restrictions are not enabled",

			command:              "get pod -n team-a",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
			},
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
				AllowedKubectlResourceMap: map[string]bool{
					"pod": true,
				},
			},

			expKubectlExecuted: true,
			expOutMsg:          "Cluster: test\nkubectl executed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			cfg := fixCfgWithKubectlExecutor(t, tc.kubectlCfg)

			wasKubectlExecuted := false
			executor := NewKubectl(logger, cfg, tc.resMapping, func(command string, args []string) (string, error) {
				wasKubectlExecuted = true
				return "kubectl executed", nil
			})

			// when
			canHandle := executor.CanHandle(strings.Fields(strings.TrimSpace(tc.command)))
			gotOutMsg, err := executor.Execute(tc.command, !tc.channelNotAuthorized)

			// then
			assert.True(t, canHandle)
			require.NoError(t, err)
			assert.Equal(t, tc.expKubectlExecuted, wasKubectlExecuted)
			assert.Equal(t, tc.expOutMsg, gotOutMsg)
		})
	}
}

func TestKubectlCanHandle(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	tests := []struct {
		name string

		command      string
		expCanHandle bool
		resMapping   ResourceMapping
	}{
		{
			name:    "Should allow for known verb",
			command: "get pod --cluster-name test",
			resMapping: ResourceMapping{
				AllowedKubectlVerbMap: map[string]bool{
					"get": true,
				},
			},

			expCanHandle: true,
		},
		{
			name:       "Should forbid if verbs is unknown",
			command:    "get pod --cluster-name test",
			resMapping: ResourceMapping{},

			expCanHandle: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			executor := NewKubectl(logger, config.Config{}, tc.resMapping, nil)

			// when
			canHandle := executor.CanHandle(strings.Fields(strings.TrimSpace(tc.command)))

			// then
			assert.Equal(t, tc.expCanHandle, canHandle)
		})
	}
}

func fixCfgWithKubectlExecutor(t *testing.T, executor config.Kubectl) config.Config {
	t.Helper()

	return config.Config{
		Settings: config.Settings{
			ClusterName: "test",
		},
		Executors: config.IndexableMap[config.Executors]{
			"default": config.Executors{
				Kubectl: executor,
			},
		},
	}
}
