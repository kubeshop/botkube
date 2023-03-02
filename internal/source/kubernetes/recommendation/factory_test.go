package recommendation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
	"github.com/kubeshop/botkube/pkg/ptr"
)

func TestFactory_New(t *testing.T) {
	// given

	cfg := config.Config{
		Recommendations: &config.Recommendations{
			Pod: config.PodRecommendations{
				LabelsSet:        ptr.Bool(true),
				NoLatestImageTag: ptr.Bool(false),
			},
			Ingress: config.IngressRecommendations{
				BackendServiceValid: ptr.Bool(true),
				// keep TLSSecretValid not specified
			},
		},
	}
	expectedNames := []string{
		"PodLabelsSet",
		"IngressBackendServiceValid",
	}
	expectedRecCfg := config.Recommendations{
		Pod: config.PodRecommendations{
			NoLatestImageTag: ptr.Bool(false),
			LabelsSet:        ptr.Bool(true),
		},
		Ingress: config.IngressRecommendations{
			BackendServiceValid: ptr.Bool(true),
			TLSSecretValid:      nil,
		},
	}

	factory := recommendation.NewFactory(loggerx.NewNoop(), nil)

	// when
	recRunner, recCfg := factory.New(cfg)
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
