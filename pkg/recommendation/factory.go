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
func (f *Factory) NewForSources(sources config.IndexableMap[config.Sources]) Set {
	set := make(map[string]Recommendation)
	for _, source := range sources {
		f.createIfNotExist(source.Recommendations.Pod.LabelsSet, set,
			func() Recommendation { return NewPodLabelsSet() },
		)
		f.createIfNotExist(source.Recommendations.Pod.NoLatestImageTag, set,
			func() Recommendation { return NewPodNoLatestImageTag() },
		)
		f.createIfNotExist(source.Recommendations.Ingress.BackendServiceValid, set,
			func() Recommendation { return NewIngressBackendServiceValid(f.dynamicCli) },
		)
		f.createIfNotExist(source.Recommendations.Ingress.TLSSecretValid, set,
			func() Recommendation { return NewIngressTLSSecretValid(f.dynamicCli) },
		)
	}
	return newRecommendationsSet(f.logger, set)
}

func (f *Factory) createIfNotExist(condition bool, set map[string]Recommendation, constructorFn func() Recommendation) {
	if !condition {
		return
	}

	// this solution has a downside that we need to create an instance every time,
	// but it seems to be the least error-prone way to do it.
	// Alternative: provide key names manually.
	recomm := constructorFn()

	key := recomm.Name()
	if _, ok := set[key]; ok {
		return
	}

	set[key] = recomm
}
