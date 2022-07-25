package filterengine

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/filterengine/filters"
)

const (
	filterLogFieldKey    = "filter"
	componentLogFieldKey = "component"
)

// WithAllFilters returns new DefaultFilterEngine instance with all filters registered.
func WithAllFilters(logger *logrus.Logger, dynamicCli dynamic.Interface, mapper meta.RESTMapper, conf *config.Config) *DefaultFilterEngine {
	enabledFilters := []Filter{
		filters.NewImageTagChecker(logger.WithField(filterLogFieldKey, "Image Tag Checker")),
		filters.NewIngressValidator(logger.WithField(filterLogFieldKey, "Ingress Validator"), dynamicCli),
		filters.NewObjectAnnotationChecker(logger.WithField(filterLogFieldKey, "Object Annotation Checker"), dynamicCli, mapper),
		filters.NewPodLabelChecker(logger.WithField(filterLogFieldKey, "Pod Label Checker"), dynamicCli, mapper),
		filters.NewNodeEventsChecker(logger.WithField(filterLogFieldKey, "Node Events Checker")),
	}

	if len(conf.Sources) > 0 {
		res := conf.Sources.GetFirst().Kubernetes.Resources
		enabledFilters = append(enabledFilters, filters.NewNamespaceChecker(logger.WithField(filterLogFieldKey, "Namespace Checker"), res))
	}

	filterEngine := New(logger.WithField(componentLogFieldKey, "Filter Engine"))
	filterEngine.Register(enabledFilters...)

	return filterEngine
}
