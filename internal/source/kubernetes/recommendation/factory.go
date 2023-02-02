package recommendation

import (
	"context"
	"github.com/kubeshop/botkube/internal/source/kubernetes/config"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"

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
func (f *Factory) NewForSources(cfg config.Config) (AggregatedRunner, config.Recommendations) {
	recommendations := f.recommendationsForConfig(cfg.Recommendations)
	return newAggregatedRunner(f.logger, recommendations), cfg.Recommendations
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
