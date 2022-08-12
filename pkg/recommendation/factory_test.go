package recommendation_test

import (
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/ptr"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

func TestFactory_NewForSources(t *testing.T) {
	// given
	sources := map[string]config.Sources{
		"first": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						LabelsSet:        ptr.Bool(true),
						NoLatestImageTag: ptr.Bool(false),
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: ptr.Bool(false),
						// keep TLSSecretValid not specified
					},
				},
			},
		},
		"second": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						// keep LabelsSet not specified
						NoLatestImageTag: ptr.Bool(true), // override `false` from `second`
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: ptr.Bool(false),
						TLSSecretValid:      ptr.Bool(true),
					},
				},
			},
		},
		"third": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						NoLatestImageTag: ptr.Bool(false), // override `true` from `second`
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: ptr.Bool(true), // override `false` from `first`
						// keep TLSSecretValid not specified
					},
				},
			},
		},
	}

	mapKeyOrder := []string{"first", "second", "third"}

	expected := map[string]struct{}{
		"PodLabelsSet":               {},
		"IngressTLSSecretValid":      {},
		"IngressBackendServiceValid": {},
	}
	logger, _ := logtest.NewNullLogger()
	factory := recommendation.NewFactory(logger, nil)

	// when
	res := factory.NewForSources(sources, mapKeyOrder)
	actual := res.Set()

	// then
	require.Len(t, actual, len(expected))
	for key := range expected {
		val, ok := actual[key]
		require.True(t, ok)
		require.NotNil(t, val)

		assert.Equal(t, key, val.Name())
	}
}
