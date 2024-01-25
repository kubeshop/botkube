package recommendation

import (
	"context"
	"github.com/kubeshop/botkube/pkg/ptr"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
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

// New creates a new AggregatedRunner.
func (f *Factory) New(cfg config.Config) (AggregatedRunner, config.Recommendations) {
	recommendations := f.recommendationsForConfig(*cfg.Recommendations)
	return newAggregatedRunner(f.logger, recommendations), *cfg.Recommendations
}

func (f *Factory) recommendationsForConfig(cfg config.Recommendations) []Recommendation {
	var recommendations []Recommendation
	if ptr.ToValue(cfg.Pod.LabelsSet) {
		recommendations = append(recommendations, NewPodLabelsSet())
	}

	if ptr.ToValue(cfg.Pod.NoLatestImageTag) {
		recommendations = append(recommendations, NewPodNoLatestImageTag())
	}

	if ptr.ToValue(cfg.Ingress.BackendServiceValid) {
		recommendations = append(recommendations, NewIngressBackendServiceValid(f.dynamicCli))
	}

	if ptr.ToValue(cfg.Ingress.TLSSecretValid) {
		recommendations = append(recommendations, NewIngressTLSSecretValid(f.dynamicCli))
	}

	return recommendations
}
