package config

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/config/remote"
)

var _ DeploymentClient = &fakeGqlClient{}

type fakeGqlClient struct {
}

func (f *fakeGqlClient) GetConfigWithResourceVersion(context.Context) (remote.Deployment, error) {
	return remote.Deployment{
		ResourceVersion: 3,
		YAMLConfig:      "communications:\n  default-group:\n    socketSlack:\n      appToken: xapp-1-A047D1ZJ03B-4262138376928\n      botToken: xoxb-3933899240838\n      channels:\n        botkube-demo:\n          bindings:\n            executors:\n            - kubectl-read-only\n            sources:\n            - kubernetes-info\n          name: botkube-demo\n          notification:\n            disabled: false\n      enabled: true\nexecutors:\n  kubectl-read-only:\n    kubectl:\n      commands:\n        resources:\n        - deployments\n        - pods\n        - namespaces\n        - daemonsets\n        - statefulsets\n        - storageclasses\n        - nodes\n        verbs:\n        - api-resources\n        - api-versions\n        - cluster-info\n        - describe\n        - diff\n        - explain\n        - get\n        - logs\n        - top\n        - auth\n      defaultNamespace: default\n      enabled: true\n      namespaces:\n        include:\n        - .*\n      restrictAccess: false\nsettings:\n  clusterName: qa\nsources:\n  kubernetes-info:\n    displayName: Kubernetes Information\n    kubernetes:\n      recommendations:\n        ingress:\n          backendServiceValid: true\n          tlsSecretValid: true\n        pod:\n          labelsSet: true\n          noLatestImageTag: true\n",
	}, nil
}

func TestGqlProviderSuccess(t *testing.T) {
	//given
	f := fakeGqlClient{}
	p := NewGqlProvider(&f)
	expected, err := os.ReadFile("testdata/TestGqlProviderSuccess/config.yaml")
	require.NoError(t, err)

	// when
	configs, ver, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	assert.Equal(t, string(expected), string(configs[0]))
	assert.Equal(t, 3, ver)
}
