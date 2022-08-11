package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

var rawExecutorsConfig = `
executors:
  'kubectl-team-a':
    kubectl:
      enabled: true
      namespaces:
        include: [ "team-a" ]
      commands:
        verbs: [ "get" ]
        resources: [ "deployments" ]
      defaultNamespace: "team-a"
  'kubectl-team-b':
    kubectl:
      enabled: true
      namespaces:
        include: [ "team-b" ]
      commands:
        verbs: [ "get", "describe" ]
        resources: [ "deployments", "pods" ]
  'kubectl-global':
    kubectl:
      enabled: true
      namespaces:
        include: [ ".*" ]
      commands:
        verbs: [ "logs", "top" ]
        resources: [ ]
  'kubectl-exec':
    kubectl:
      enabled: false
      namespaces:
        include: [ ".*" ]
      commands:
        verbs: [ "exec" ]
        resources: [ ]`

func TestDefaultExecutor_getEnabledKubectlConfigs(t *testing.T) {
	testCases := []struct {
		name            string
		executorsConfig string
		bindings        []string

		expOutput string
	}{
		{
			name:            "All bindings specified",
			executorsConfig: rawExecutorsConfig,
			bindings: []string{
				"kubectl-team-a",
				"kubectl-team-b",
				"kubectl-global",
				"kubectl-exec",
			},
			expOutput: heredoc.Doc(`
           Enabled executors:
             kubectl:
               kubectl-global:
                 namespaces:
                   include:
                     - .*
                 enabled: true
                 commands:
                   verbs:
                     - logs
                     - top
                   resources: []
               kubectl-team-a:
                 namespaces:
                   include:
                     - team-a
                 enabled: true
                 commands:
                   verbs:
                     - get
                   resources:
                     - deployments
                 defaultNamespace: team-a
               kubectl-team-b:
                 namespaces:
                   include:
                     - team-b
                 enabled: true
                 commands:
                   verbs:
                     - get
                     - describe
                   resources:
                     - deployments
                     - pods
           `),
		},
		{
			name:            "One enabled binding specified",
			executorsConfig: rawExecutorsConfig,
			bindings: []string{
				"kubectl-team-a",
			},
			expOutput: heredoc.Doc(`
           Enabled executors:
             kubectl:
               kubectl-team-a:
                 namespaces:
                   include:
                     - team-a
                 enabled: true
                 commands:
                   verbs:
                     - get
                   resources:
                     - deployments
                 defaultNamespace: team-a
           `),
		},
		{
			name:            "One disabled binding specified",
			executorsConfig: rawExecutorsConfig,
			bindings: []string{
				"kubectl-exec",
			},
			expOutput: heredoc.Doc(`
           Enabled executors:
             kubectl: {}
           `),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			executors := fixExecutorsConfig(t, tc.executorsConfig)
			executor := &DefaultExecutor{
				merger:   kubectl.NewMerger(executors),
				bindings: tc.bindings,
			}

			// when
			res, err := executor.getEnabledKubectlExecutorsInChannel()

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, res)
		})
	}
}

func fixExecutorsConfig(t *testing.T, raw string) map[string]config.Executors {
	t.Helper()

	var givenCfg config.Config
	err := yaml.Unmarshal([]byte(raw), &givenCfg)
	require.NoError(t, err)

	return givenCfg.Executors
}
