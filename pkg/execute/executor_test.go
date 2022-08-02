package execute

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
)

var rawEnabledExecutorsConfig = `
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
        include: [ "all" ]
      commands:
        verbs: [ "logs", "top" ]
        resources: [ ]
  'kubectl-exec':
    kubectl:
      enabled: false
      namespaces:
        include: [ "all" ]
      commands:
        verbs: [ "exec" ]
        resources: [ ]`

var rawDisabledExecutorsConfig = `
executors:
  'kubectl-team-a':
    kubectl:
      enabled: false
      namespaces:
        include: [ "team-a" ]
      commands:
        verbs: [ "get" ]
        resources: [ "deployments" ]
      defaultNamespace: "team-a"
  'kubectl-team-b':
    kubectl:
      enabled: false
      namespaces:
        include: [ "team-b" ]
      commands:
        verbs: [ "get", "describe" ]
        resources: [ "deployments", "pods" ]`

func TestDefaultExecutor_getSortedNamespaceConfig(t *testing.T) {
	testCases := []struct {
		name            string
		executorsConfig string
		expOutput       string
	}{
		{
			name:            "All namespaces enabled",
			executorsConfig: rawEnabledExecutorsConfig,
			expOutput: heredoc.Doc(`
           enabled:
             kubectl:
               kubectl-global:
                 namespaces:
                   include:
                     - all
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
			name:            "All namespace enabled and a few ignored",
			executorsConfig: rawDisabledExecutorsConfig,
			expOutput: heredoc.Doc(`
           enabled:
             kubectl: {}
           `),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			executor := &DefaultExecutor{
				cfg: config.Config{
					Executors: fixExecutorsConfig(t, tc.executorsConfig),
				},
			}

			// when
			res, err := executor.getEnabledKubectlConfigs()

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expOutput, res)
		})
	}
}

func fixExecutorsConfig(t *testing.T, raw string) config.IndexableMap[config.Executors] {
	t.Helper()

	var givenCfg config.Config
	err := yaml.Unmarshal([]byte(raw), &givenCfg)
	require.NoError(t, err)

	return givenCfg.Executors
}
