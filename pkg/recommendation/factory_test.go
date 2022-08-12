package recommendation_test

import (
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

func TestFactory_NewForSources(t *testing.T) {
	// given
	sources := map[string]config.Sources{
		"first": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						LabelsSet:        true,
						NoLatestImageTag: false,
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: false,
						TLSSecretValid:      false,
					},
				},
			},
		},
		"second": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						LabelsSet:        true,
						NoLatestImageTag: true,
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: false,
						TLSSecretValid:      true,
					},
				},
			},
		},
		"third": {
			Kubernetes: config.KubernetesSource{
				Recommendations: config.Recommendations{
					Pod: config.PodRecommendations{
						LabelsSet:        false,
						NoLatestImageTag: true,
					},
					Ingress: config.IngressRecommendations{
						BackendServiceValid: true,
						TLSSecretValid:      true,
					},
				},
			},
		},
	}
	expected := map[string]struct{}{
		"PodNoLatestImageTag":        {},
		"PodLabelsSet":               {},
		"IngressTLSSecretValid":      {},
		"IngressBackendServiceValid": {},
	}
	logger, _ := logtest.NewNullLogger()
	factory := recommendation.NewFactory(logger, nil)

	// when
	res := factory.NewForSources(sources)
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
