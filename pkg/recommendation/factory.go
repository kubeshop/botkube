package recommendation

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// Recommendation performs checks for a given event.
type Recommendation interface {
	Do(ctx context.Context, event event.Event) (Result, error)
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

// NewForSources merges recommendation options from multiple sources, and creates a new AggregatedRunner.
func (f *Factory) NewForSources(sources map[string]config.Sources, mapKeyOrder []string) (AggregatedRunner, config.Recommendations) {
	mergedCfg := f.mergeConfig(sources, mapKeyOrder)
	recommendations := f.recommendationsForConfig(mergedCfg)
	return newAggregatedRunner(f.logger, recommendations), mergedCfg
}

func (f *Factory) mergeConfig(sources map[string]config.Sources, mapKeyOrder []string) config.Recommendations {
	mergedCfg := config.Recommendations{}
	for _, key := range mapKeyOrder {
		source, exists := sources[key]
		if !exists {
			continue
		}

		sourceCfg := source.Kubernetes.Recommendations
		if sourceCfg.Pod.LabelsSet != nil {
			mergedCfg.Pod.LabelsSet = sourceCfg.Pod.LabelsSet
		}
		if sourceCfg.Pod.NoLatestImageTag != nil {
			mergedCfg.Pod.NoLatestImageTag = sourceCfg.Pod.NoLatestImageTag
		}
		if sourceCfg.Ingress.BackendServiceValid != nil {
			mergedCfg.Ingress.BackendServiceValid = sourceCfg.Ingress.BackendServiceValid
		}
		if sourceCfg.Ingress.TLSSecretValid != nil {
			mergedCfg.Ingress.TLSSecretValid = sourceCfg.Ingress.TLSSecretValid
		}
	}

	return mergedCfg
}

func (f *Factory) recommendationsForConfig(cfg config.Recommendations) []Recommendation {
	var recommendations []Recommendation
	if ptr.IsTrue(cfg.Pod.LabelsSet) {
		recommendations = append(recommendations, NewPodLabelsSet())
	}

	if ptr.IsTrue(cfg.Pod.NoLatestImageTag) {
		recommendations = append(recommendations, NewPodNoLatestImageTag())
	}

	if ptr.IsTrue(cfg.Ingress.BackendServiceValid) {
		recommendations = append(recommendations, NewIngressBackendServiceValid(f.dynamicCli))
	}

	if ptr.IsTrue(cfg.Ingress.TLSSecretValid) {
		recommendations = append(recommendations, NewIngressTLSSecretValid(f.dynamicCli))
	}

	return recommendations
}
