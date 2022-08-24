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
	expectedNames := []string{
		"PodLabelsSet",
		"IngressBackendServiceValid",
		"IngressTLSSecretValid",
	}
	expectedRecCfg := config.Recommendations{
		Pod: config.PodRecommendations{
			NoLatestImageTag: ptr.Bool(false),
			LabelsSet:        ptr.Bool(true),
		},
		Ingress: config.IngressRecommendations{
			BackendServiceValid: ptr.Bool(true),
			TLSSecretValid:      ptr.Bool(true),
		},
	}

	logger, _ := logtest.NewNullLogger()
	factory := recommendation.NewFactory(logger, nil)

	// when
	recRunner, recCfg := factory.NewForSources(sources, mapKeyOrder)
	actualRecomms := recRunner.Recommendations()

	// then
	assert.Equal(t, expectedRecCfg, recCfg)
	require.Len(t, actualRecomms, len(expectedNames))

	var actualNames []string
	for _, r := range actualRecomms {
		actualNames = append(actualNames, r.Name())
	}

	assert.Equal(t, expectedNames, actualNames)
}
