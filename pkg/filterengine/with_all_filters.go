package filterengine

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/filterengine/filters"
)

const filterLogFieldKey = "filter"
const componentLogFieldKey = "component"

// WithAllFilters returns new DefaultFilterEngine instance with all filters registered.
func WithAllFilters(logger *logrus.Logger, dynamicCli dynamic.Interface, mapper meta.RESTMapper, conf *config.Config) *DefaultFilterEngine {
	filterEngine := New(logger.WithField(componentLogFieldKey, "Filter Engine"))
	filterEngine.Register([]Filter{
		filters.NewImageTagChecker(logger.WithField(filterLogFieldKey, "Image Tag Checker")),
		filters.NewIngressValidator(logger.WithField(filterLogFieldKey, "Ingress Validator"), dynamicCli),
		filters.NewObjectAnnotationChecker(logger.WithField(filterLogFieldKey, "Object Annotation Checker"), dynamicCli, mapper),
		filters.NewPodLabelChecker(logger.WithField(filterLogFieldKey, "Pod Label Checker"), dynamicCli, mapper),
		filters.NewNamespaceChecker(logger.WithField(filterLogFieldKey, "Namespace Checker"), conf.Resources),
		filters.NewNodeEventsChecker(logger.WithField(filterLogFieldKey, "Node Events Checker")),
	}...)

	return filterEngine
}
