package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ GqlClient = &fakeGqlClient{}

type fakeGqlClient struct {
}

func (f *fakeGqlClient) GetDeployment(ctx context.Context, id string) (Deployment, error) {
	return Deployment{
		BotkubeConfig: "{\"settings\":{\"clusterName\":\"qa\"},\"sources\":{\"kubernetes-info\":{\"displayName\":\"Kubernetes Information\",\"kubernetes\":{\"recommendations\":{\"pod\":{\"noLatestImageTag\":true,\"labelsSet\":true},\"ingress\":{\"backendServiceValid\":true,\"tlsSecretValid\":true}}}}},\"executors\":{\"kubectl-read-only\":{\"kubectl\":{\"namespaces\":{\"include\":[\".*\"]},\"enabled\":true,\"commands\":{\"verbs\":[\"api-resources\",\"api-versions\",\"cluster-info\",\"describe\",\"diff\",\"explain\",\"get\",\"logs\",\"top\",\"auth\"],\"resources\":[\"deployments\",\"pods\",\"namespaces\",\"daemonsets\",\"statefulsets\",\"storageclasses\",\"nodes\"]},\"defaultNamespace\":\"default\",\"restrictAccess\":false}}},\"communications\":{\"default-group\":{\"socketSlack\":{\"enabled\":true,\"channels\":{\"botkube-demo\":{\"name\":\"botkube-demo\",\"notification\":{\"disabled\":false},\"bindings\":{\"sources\":[\"kubernetes-info\"],\"executors\":[\"kubectl-read-only\"]}}},\"botToken\":\"xoxb-3933899240838-4238355342083-2rUu0vkKDECUj27qECYPCqGd\",\"appToken\":\"xapp-1-A047D1ZJ03B-4262138376928-33d80c4792bb44c189b4f2948e0a00351316b3f694332fdec1ecdc61046320ab\"}}}}",
	}, nil
}

func TestGqlProviderSuccess(t *testing.T) {
	//given
	f := fakeGqlClient{}
	t.Setenv("CONFIG_SOURCE_IDENTIFIER", "16")
	p := NewGqlProvider(&f)

	// when
	configs, err := p.Configs(context.Background())

	// then
	assert.NoError(t, err)
	assert.Contains(t, configs[0], "/tmp/botkube-")
}
