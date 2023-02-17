package filterengine

import (
	config2 "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/filterengine/filters"
)

const (
	filterLogFieldKey    = "filter"
	componentLogFieldKey = "component"
)

// WithAllFilters returns new DefaultFilterEngine instance with all filters registered.
func WithAllFilters(logger logrus.FieldLogger, dynamicCli dynamic.Interface, mapper meta.RESTMapper, cfg *config2.Filters) *DefaultFilterEngine {
	filterEngine := New(logger.WithField(componentLogFieldKey, "Filter Engine"))
	filterEngine.Register([]RegisteredFilter{
		{
			Filter:  filters.NewObjectAnnotationChecker(logger.WithField(filterLogFieldKey, "Object Annotation Checker"), dynamicCli, mapper),
			Enabled: cfg.Kubernetes.ObjectAnnotationChecker,
		},
		{
			Filter:  filters.NewNodeEventsChecker(logger.WithField(filterLogFieldKey, "Node Events Checker")),
			Enabled: cfg.Kubernetes.NodeEventsChecker,
		},
	}...)

	return filterEngine
}
