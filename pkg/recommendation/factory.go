package recommendation

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// Recommendation performs checks for a given event.
type Recommendation interface {
	Do(ctx context.Context, event events.Event) (Result, error)
	Name() string
}

// Result is the result of a recommendation check.
type Result struct {
	Info     []string
	Warnings []string
}

// Factory is a factory for creating recommendation sets.
type Factory struct {
	logger     logrus.FieldLogger
	dynamicCli dynamic.Interface
}

// NewFactory creates a new Factory instance.
func NewFactory(logger logrus.FieldLogger, dynamicCli dynamic.Interface) *Factory {
	return &Factory{logger: logger, dynamicCli: dynamicCli}
}

// NewForSources merges recommendation options from multiple sources, and creates a new Set.
func (f *Factory) NewForSources(sources map[string]config.Sources, mapKeyOrder []string) Set {
	set := make(map[string]Recommendation)
	for _, key := range mapKeyOrder {
		source, exists := sources[key]
		if !exists {
			continue
		}

		k8sSource := source.Kubernetes
		f.enableRecommendationIfShould(k8sSource.Recommendations.Pod.LabelsSet, set,
			func() Recommendation { return NewPodLabelsSet() },
		)
		f.enableRecommendationIfShould(k8sSource.Recommendations.Pod.NoLatestImageTag, set,
			func() Recommendation { return NewPodNoLatestImageTag() },
		)
		f.enableRecommendationIfShould(k8sSource.Recommendations.Ingress.BackendServiceValid, set,
			func() Recommendation { return NewIngressBackendServiceValid(f.dynamicCli) },
		)
		f.enableRecommendationIfShould(k8sSource.Recommendations.Ingress.TLSSecretValid, set,
			func() Recommendation { return NewIngressTLSSecretValid(f.dynamicCli) },
		)
	}
	return newRecommendationsSet(f.logger, set)
}

func (f *Factory) enableRecommendationIfShould(condition *bool, set map[string]Recommendation, constructorFn func() Recommendation) {
	if condition == nil {
		// not specified = keep previous configuration
		return
	}

	recomm := constructorFn()

	if !*condition {
		// Disabled - remove from set if exists
		delete(set, recomm.Name())
		return
	}

	// Enabled - set if it doesn't exist
	key := recomm.Name()
	if _, ok := set[key]; ok {
		return
	}

	set[key] = recomm
}
