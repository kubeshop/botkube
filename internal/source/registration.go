package source

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/k8sutil"
	"github.com/kubeshop/botkube/pkg/multierror"
)

type registration struct {
	informer        cache.SharedIndexInformer
	log             logrus.FieldLogger
	mapper          meta.RESTMapper
	dynamicCli      dynamic.Interface
	events          []config.EventType
	mappedResources []string
	mappedEvent     config.EventType
}

func (r registration) handleEvent(ctx context.Context, resource string, eventType config.EventType, sourceRoutes []route, fn eventHandler) {
	handleFunc := func(oldObj, newObj interface{}) {
		logger := r.log.WithFields(logrus.Fields{
			"eventHandler": eventType,
			"resource":     resource,
			"object":       newObj,
		})

		event, err := r.eventForObj(ctx, newObj, eventType, resource)
		if err != nil {
			logger.Errorf("while creating new event: %s", err.Error())
			return
		}

		sources, diffs, err := r.qualifySourcesForEvent(event, newObj, oldObj, sourceRoutes)
		if err != nil {
			logger.Errorf("while getting sources for event: %s", err.Error())
			// continue anyway, there could be still some sources to handle
		}
		if len(sources) == 0 {
			return
		}
		fn(ctx, event, sources, diffs)
	}

	var resourceEventHandlerFuncs cache.ResourceEventHandlerFuncs
	switch eventType {
	case config.CreateEvent:
		resourceEventHandlerFuncs.AddFunc = func(obj interface{}) { handleFunc(nil, obj) }
	case config.DeleteEvent:
		resourceEventHandlerFuncs.DeleteFunc = func(obj interface{}) { handleFunc(nil, obj) }
	case config.UpdateEvent:
		resourceEventHandlerFuncs.UpdateFunc = handleFunc
	}

	r.informer.AddEventHandler(resourceEventHandlerFuncs)
}

func (r registration) handleMapped(ctx context.Context, eventType config.EventType, routeTable map[string][]entry, fn eventHandler) {
	r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var eventObj coreV1.Event
			err := k8sutil.TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
			if err != nil {
				r.log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(obj), reflect.TypeOf(eventObj))
				return
			}
			_, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				r.log.Errorf("Failed to get MetaNamespaceKey from event resource")
				return
			}

			// Find involved object type
			gvr, err := k8sutil.GetResourceFromKind(r.mapper, eventObj.InvolvedObject.GroupVersionKind())
			if err != nil {
				r.log.Errorf("Failed to get involved object: %v", err)
				return
			}

			if !r.canHandleEvent(eventObj.Type) {
				return
			}

			gvrString := gvrToString(gvr)
			if !r.includesSrcResource(gvrString) {
				return
			}

			event, err := r.eventForObj(ctx, obj, eventType, gvrString)
			if err != nil {
				r.log.Errorf("while creating new event: %s", err.Error())
				return
			}

			sourceRoutes := sourceRoutes(routeTable, gvrString, eventType)
			sources, err := r.sourcesForEvent(sourceRoutes, event)
			if err != nil {
				r.log.Errorf("cannot calculate sources for observed mapped resource event: %q in Add event handler: %s", eventType, err.Error())
				// continue anyway, there could be still some sources to handle
			}
			if len(sources) == 0 {
				return
			}
			fn(ctx, event, sources, nil)
		},
	})
}

func (r registration) canHandleEvent(target string) bool {
	for _, e := range r.events {
		if strings.EqualFold(target, e.String()) {
			return true
		}
	}
	return false
}

func (r registration) includesSrcResource(resource string) bool {
	for _, src := range r.mappedResources {
		if src == resource {
			return true
		}
	}
	return false
}

func (r registration) sourcesForEvent(routes []route, event event.Event) ([]string, error) {
	var out []string

	r.log.WithField("event", event).WithField("routes", routes).Debugf("handling event")

	errs := multierror.New()
	for _, route := range routes {
		// event reason
		match, err := matchRegexForStringIfDefined(route.event.Reason, event.Reason)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if !match {
			r.log.Debugf("Ignoring as reason %q doesn't match regex %q", event.Reason, route.event.Reason)
			continue
		}

		// event message
		match, err = matchRegexForStringsIfDefined(route.event.Message, event.Messages)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if !match {
			r.log.Debugf("Ignoring as messages %q don't match regex %q", strings.Join(event.Messages, ";"), route.event.Message)
			continue
		}

		// resource name
		match, err = matchRegexForStringIfDefined(route.resourceName, event.Name)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		if !match {
			r.log.Debugf("Ignoring as resource name %q doesn't match regex %q", event.Name, route.resourceName)
			continue
		}

		// namespace
		if event.Namespace != "" && !route.namespaces.IsAllowed(event.Namespace) {
			continue
		}

		// annotations
		if !kvsSatisfiedForMap(route.annotations, event.ObjectMeta.Annotations) {
			continue
		}

		// labels
		if !kvsSatisfiedForMap(route.labels, event.ObjectMeta.Labels) {
			continue
		}

		out = append(out, route.source)
	}

	return out, errs.ErrorOrNil()
}

func matchRegexForStringIfDefined(regexStr, str string) (bool, error) {
	return matchRegexForStringsIfDefined(regexStr, []string{str})
}

func matchRegexForStringsIfDefined(regexStr string, str []string) (bool, error) {
	if regexStr == "" {
		return true, nil
	}

	regex, err := regexp2.Compile(regexStr, regexp2.None)
	if err != nil {
		return false, fmt.Errorf("while compiling regex: %w", err)
	}

	if len(str) == 0 {
		// no messages, so let's check if regex matches empty string
		str = append(str, "")
	}

	errs := multierror.New()
	for _, s := range str {
		match, err := regex.MatchString(s)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while matching regex %s: %w", regexStr, err))
			continue
		}

		if match {
			return true, errs.ErrorOrNil()
		}
	}

	return false, errs.ErrorOrNil()
}

func kvsSatisfiedForMap(expectedKV, obj map[string]string) bool {
	if len(expectedKV) == 0 {
		return true
	}

	if len(obj) == 0 {
		return false
	}

	for k, v := range expectedKV {
		got, ok := obj[k]
		if !ok {
			return false
		}

		if got != v {
			return false
		}
	}

	return true
}

func (r registration) eventForObj(ctx context.Context, obj interface{}, eventType config.EventType, resource string) (event.Event, error) {
	objectMeta, err := k8sutil.GetObjectMetaData(ctx, r.dynamicCli, r.mapper, obj)
	if err != nil {
		return event.Event{}, fmt.Errorf("while getting object metadata: %s", err.Error())
	}

	e, err := event.New(objectMeta, obj, eventType, resource)
	if err != nil {
		return event.Event{}, fmt.Errorf("while creating new event: %s", err.Error())
	}

	return e, nil
}

func (r registration) qualifySourcesForEvent(
	event event.Event,
	newObj, oldObj interface{},
	routes []route,
) ([]string, []string, error) {
	candidates, err := r.sourcesForEvent(routes, event)
	if err != nil {
		return nil, nil, fmt.Errorf("while getting sources for event: %w", err)
	}

	if event.Type == config.UpdateEvent {
		return r.qualifySourcesForUpdate(newObj, oldObj, routes, candidates)
	}

	return candidates, nil, nil
}

func (r registration) qualifySourcesForUpdate(
	newObj, oldObj interface{},
	routes []route,
	candidates []string,
) ([]string, []string, error) {
	var sources, diffs []string

	var oldUnstruct, newUnstruct *unstructured.Unstructured
	var ok bool

	if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
		r.log.Error("Failed to typecast old object to Unstructured.")
	}

	if newUnstruct, ok = newObj.(*unstructured.Unstructured); !ok {
		r.log.Error("Failed to typecast new object to Unstructured.")
	}

	r.log.Debugf("qualifySourcesForUpdate source candidates: %+v", candidates)

	for _, source := range candidates {
		for _, route := range routes {
			if route.source != source {
				continue
			}

			if !route.hasActionableUpdateSetting() {
				r.log.Debugf("Qualified for update: source: %s, with no updateSettings set", source)
				sources = append(sources, source)
				continue
			}

			diff, err := k8sutil.Diff(oldUnstruct.Object, newUnstruct.Object, route.updateSetting)
			if err != nil {
				r.log.Errorf("while getting diff: %w", err)
			}
			r.log.Debugf("About to qualify source: %s for update, diff: %s, updateSetting: %+v", source, diff, route.updateSetting)

			if len(diff) > 0 && route.updateSetting.IncludeDiff {
				sources = append(sources, source)
				diffs = append(diffs, diff)
				r.log.Debugf("Qualified for update: source: %s for update, diff: %s, updateSetting: %+v", source, diff, route.updateSetting)
			}
		}
	}

	return sources, diffs, nil
}

// gvrToString converts GVR formats to string.
func gvrToString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}
