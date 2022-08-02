package kubectl_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/config"
)

var RawExecutorsConfig = `
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

func fixExecutorsConfig(t *testing.T) config.IndexableMap[config.Executors] {
	t.Helper()

	var givenCfg config.Config
	err := yaml.Unmarshal([]byte(RawExecutorsConfig), &givenCfg)
	require.NoError(t, err)

	return givenCfg.Executors
}
