package execute

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/ptr"
)

func TestKubectlExecuteErrors(t *testing.T) {
	tests := []struct {
		name string

		command              string
		clusterName          string
		channelNotAuthorized bool
		kubectlCfg           config.Kubectl
		expKubectlExecuted   bool
		expErr               string
	}{
		{
			name: "Should forbid execution from not authorized channel when restrictions are enabled",

			command:              "get pod --cluster-name test",
			clusterName:          "test",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"default"},
				},
				RestrictAccess: ptr.Bool(true),
				Commands: config.Commands{
					Verbs: []string{"get"},
				},
			},

			expErr: "Sorry, this channel is not authorized to execute kubectl command on cluster 'test'.",
		},
		{
			name: "Should forbid execution if resource is not allowed in config",

			command: "get pod -n foo",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"foo"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: nil,
				},
			},
			expErr: "Sorry, the kubectl command is not authorized to work with 'pod' resources in the 'foo' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should forbid execution if namespace is not allowed in config",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: nil, // no namespace allowed.
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed in the 'default' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should use default Namespace from config if not specified in command",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled:          true,
				DefaultNamespace: "from-config",
				Namespaces: config.Namespaces{
					Include: nil, // forbid `from-config` to get a suitable error message.
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed in the 'from-config' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should explicitly use 'default' Namespace if not specified both in command and config",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: nil, // forbid `default` to get a suitable error message.
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed in the 'default' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should forbid execution in not allowed namespace",

			command: "get pod -n team-b",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed in the 'team-b' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should forbid execution if all namespace are allowed but command namespace is explicitly ignored in config",

			command: "get pod -n team-b",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{config.AllNamespaceIndicator},
					Exclude: []string{"team-b"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed in the 'team-b' Namespace on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Should forbid execution for all Namespaces",

			command: "get pod -A",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expErr: "Sorry, the kubectl 'get' command cannot be executed for all Namespaces on cluster 'test'. Use 'list commands' to see allowed commands.",
		},
		{
			name: "Known limitation (since v0.12.4): Return error if flag is added before resource name",

			command: "get -n team-a pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expKubectlExecuted: false,
			expErr:             "Please specify the resource name after the verb, and all flags after the resource name. Format <verb> <resource> [flags]",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			cfg := fixCfgWithKubectlExecutor(t, tc.kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)

			wasKubectlExecuted := false

			executor := NewKubectl(loggerx.NewNoop(), cfg, merger, kcChecker, cmdCombinedFunc(func(command string, args []string) (string, error) {
				wasKubectlExecuted = true
				return "kubectl executed", nil
			}))

			// when
			canHandle := executor.CanHandle(fixBindingsNames, strings.Fields(strings.TrimSpace(tc.command)))
			gotOutMsg, err := executor.Execute(fixBindingsNames, tc.command, !tc.channelNotAuthorized, CommandContext{ProvidedClusterName: tc.clusterName, ClusterName: tc.clusterName})

			// then
			assert.True(t, canHandle, "it should be able to handle the execution")
			assert.True(t, IsExecutionCommandError(err))
			assert.False(t, wasKubectlExecuted)
			assert.Empty(t, gotOutMsg)
			assert.EqualError(t, err, tc.expErr)
		})
	}
}

func TestKubectlExecute(t *testing.T) {
	tests := []struct {
		name string

		command              string
		channelNotAuthorized bool
		kubectlCfg           config.Kubectl
		expOutMsg            string
	}{
		{
			name: "Should all execution if resource is missing, so kubectl can validate it further",

			command: "get",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"default"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: nil,
				},
			},
			expOutMsg: "kubectl executed",
		},
		{
			name: "Should allow execution if verb, resource, and all namespaces are allowed",

			command: "get pod",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{config.AllNamespaceIndicator},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
		{
			name: "Should allow execution if verb, resource, and a given namespace are allowed",

			command: "get pod -n team-a",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
		{
			name: "Should allow execution from not authorized channel if restrictions are disabled",

			command:              "get pod -n team-a",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
		{
			name: "Should allow execution from not authorized channel if restrictions are disabled",

			command:              "get pod/name-foo-42 -n team-a",
			channelNotAuthorized: true,
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
		{
			name: "Should allow execution for all Namespaces",

			command: "get pod/name-foo-42 -n team-a",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
		{
			name: "Should all execution for all Namespaces",

			command: "get pod -A",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{".*"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expOutMsg: "kubectl executed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			cfg := fixCfgWithKubectlExecutor(t, tc.kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)

			wasKubectlExecuted := false

			executor := NewKubectl(loggerx.NewNoop(), cfg, merger, kcChecker, cmdCombinedFunc(func(command string, args []string) (string, error) {
				wasKubectlExecuted = true
				return "kubectl executed", nil
			}))

			// when
			canHandle := executor.CanHandle(fixBindingsNames, strings.Fields(strings.TrimSpace(tc.command)))
			gotOutMsg, err := executor.Execute(fixBindingsNames, tc.command, !tc.channelNotAuthorized, CommandContext{})

			// then
			assert.True(t, canHandle, "it should be able to handle the execution")
			require.NoError(t, err)
			assert.True(t, wasKubectlExecuted)
			assert.Equal(t, tc.expOutMsg, gotOutMsg)
		})
	}
}

func TestKubectlCanHandle(t *testing.T) {
	tests := []struct {
		name string

		command      string
		expCanHandle bool
		kubectlCfg   config.Kubectl
	}{
		{
			name:    "Should allow for known verb",
			command: "get pod --cluster-name test",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expCanHandle: true,
		},
		{
			name:    "Should allow for known verb with k8s prefix",
			command: "kubectl get pod --cluster-name test",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"get"},
					Resources: []string{"pod"},
				},
			},

			expCanHandle: true,
		},
		{
			name:    "Should forbid if verbs is unknown",
			command: "get pod --cluster-name test",
			kubectlCfg: config.Kubectl{
				Enabled: true,
				Namespaces: config.Namespaces{
					Include: []string{"team-a"},
				},
				Commands: config.Commands{
					Verbs:     []string{"describe"},
					Resources: []string{"pod"},
				},
			},

			expCanHandle: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			cfg := fixCfgWithKubectlExecutor(t, tc.kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)

			executor := NewKubectl(loggerx.NewNoop(), config.Config{}, merger, kcChecker, nil)

			// when
			canHandle := executor.CanHandle(fixBindingsNames, strings.Fields(strings.TrimSpace(tc.command)))

			// then
			assert.Equal(t, tc.expCanHandle, canHandle)
		})
	}
}

func TestKubectlGetVerb(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectedVerb string
	}{
		{
			name:         "Should get proper verb without k8s prefix",
			command:      "get pods --cluster-name test",
			expectedVerb: "get",
		},
		{
			name:         "Should get proper verb with k8s prefix kubectl",
			command:      "kubectl get pods --cluster-name test",
			expectedVerb: "get",
		},
		{
			name:         "Should get proper verb with k8s prefix kc",
			command:      "kc get pods --cluster-name test",
			expectedVerb: "get",
		},
		{
			name:         "Should get proper verb with k8s prefix k",
			command:      "k get pods --cluster-name test",
			expectedVerb: "get",
		},
	}
	kubectlCfg := config.Kubectl{
		Enabled: true,
		Namespaces: config.Namespaces{
			Include: []string{"team-a"},
		},
		Commands: config.Commands{
			Verbs:     []string{"get"},
			Resources: []string{"pod"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := fixCfgWithKubectlExecutor(t, kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)
			executor := NewKubectl(loggerx.NewNoop(), config.Config{}, merger, kcChecker, nil)

			args := strings.Fields(tc.command)
			verb := executor.GetVerb(args)

			assert.Equal(t, tc.expectedVerb, verb)
		})
	}
}

func TestKubectlGetCommandPrefix(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Should get proper command without k8s prefix",
			command:  "get pods --cluster-name test",
			expected: "get",
		},
		{
			name:     "Should get proper command with k8s prefix kubectl",
			command:  "kubectl get pods --cluster-name test",
			expected: "kubectl get",
		},
		{
			name:     "Should get proper command with k8s prefix kc",
			command:  "kc get pods --cluster-name test",
			expected: "kc get",
		},
		{
			name:     "Should get proper command with k8s prefix k",
			command:  "k get pods --cluster-name test",
			expected: "k get",
		},
	}
	kubectlCfg := config.Kubectl{
		Enabled: true,
		Namespaces: config.Namespaces{
			Include: []string{"team-a"},
		},
		Commands: config.Commands{
			Verbs:     []string{"get"},
			Resources: []string{"pod"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := fixCfgWithKubectlExecutor(t, kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)
			executor := NewKubectl(loggerx.NewNoop(), config.Config{}, merger, kcChecker, nil)

			args := strings.Fields(tc.command)
			verb := executor.GetCommandPrefix(args)

			assert.Equal(t, tc.expected, verb)
		})
	}
}

func TestKubectlGetArgsWithoutAlias(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Should get proper command without k8s prefix",
			command:  "get pods --cluster-name test",
			expected: "get pods --cluster-name test",
		},
		{
			name:     "Should get proper command with k8s prefix kubectl",
			command:  "kubectl get pods --cluster-name test",
			expected: "get pods --cluster-name test",
		},
		{
			name:     "Should get proper verb with k8s prefix kc",
			command:  "kc get pods --cluster-name test",
			expected: "get pods --cluster-name test",
		},
		{
			name:     "Should get proper verb with k8s prefix k",
			command:  "k get pods --cluster-name test",
			expected: "get pods --cluster-name test",
		},
	}
	kubectlCfg := config.Kubectl{
		Enabled: true,
		Namespaces: config.Namespaces{
			Include: []string{"team-a"},
		},
		Commands: config.Commands{
			Verbs:     []string{"get"},
			Resources: []string{"pod"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			cfg := fixCfgWithKubectlExecutor(t, kubectlCfg)
			merger := kubectl.NewMerger(cfg.Executors)
			kcChecker := kubectl.NewChecker(nil)
			executor := NewKubectl(loggerx.NewNoop(), config.Config{}, merger, kcChecker, nil)

			// when
			verb, err := executor.getArgsWithoutAlias(tc.command)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expected, strings.Join(verb, " "))
		})
	}
}

var fixBindingsNames = []string{"default"}

func fixCfgWithKubectlExecutor(t *testing.T, executor config.Kubectl) config.Config {
	t.Helper()

	return config.Config{
		Settings: config.Settings{
			ClusterName: "test",
		},
		Executors: map[string]config.Executors{
			"default": {
				Kubectl: executor,
			},
		},
	}
}

// cmdCombinedFunc type is an adapter to allow the use of ordinary functions as command handlers.
type cmdCombinedFunc func(command string, args []string) (string, error)

func (f cmdCombinedFunc) RunCombinedOutput(command string, args []string) (string, error) {
	return f(command, args)
}
