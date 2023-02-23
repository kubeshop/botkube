package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/k8sutil"
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

func (r registration) handleEvent(s Source, resource string, eventType config.EventType, routes []route, fn eventHandler) {
	handleFunc := func(oldObj, newObj interface{}) {
		logger := r.log.WithFields(logrus.Fields{
			"eventHandler": eventType,
			"resource":     resource,
			"object":       newObj,
		})

		event, err := r.eventForObj(s.ctx, newObj, eventType, resource)
		if err != nil {
			logger.Errorf("while creating new event: %s", err.Error())
			return
		}

		ok, diffs, err := r.qualifyEvent(event, newObj, oldObj, routes)
		if err != nil {
			logger.Errorf("while getting sources for event: %s", err.Error())
			// continue anyway, there could be still some sources to handle
		}
		if !ok {
			return
		}
		fn(s, event, diffs)
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

func (r registration) handleMapped(s Source, eventType config.EventType, routeTable map[string][]entry, fn eventHandler) {
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

			event, err := r.eventForObj(s.ctx, obj, eventType, gvrString)
			if err != nil {
				r.log.Errorf("while creating new event: %s", err.Error())
				return
			}

			routes := eventRoutes(routeTable, gvrString, eventType)
			ok, err := r.matchEvent(routes, event)
			if err != nil {
				r.log.Errorf("cannot calculate event for observed mapped resource event: %q in Add event handler: %s", eventType, err.Error())
				// continue anyway, there could be still some sources to handle
			}
			if !ok {
				return
			}
			fn(s, event, nil)
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

func (r registration) matchEvent(routes []route, event event.Event) (bool, error) {
	errs := multierror.New()
	for _, rt := range routes {
		// event reason
		if rt.event.Reason.AreConstraintsDefined() {
			match, err := rt.event.Reason.IsAllowed(event.Reason)
			if err != nil {
				return false, err
			}
			if !match {
				r.log.Debugf("Ignoring as reason %q doesn't match constraints %+v", event.Reason, rt.event.Reason)
				return false, nil
			}
		}

		// event message
		if rt.event.Message.AreConstraintsDefined() {
			var anyMsgMatches bool

			eventMsgs := event.Messages
			if len(eventMsgs) == 0 {
				// treat no messages as an empty message
				eventMsgs = []string{""}
			}

			for _, msg := range eventMsgs {
				match, err := rt.event.Message.IsAllowed(msg)
				if err != nil {
					return false, err
				}
				if match {
					anyMsgMatches = true
					break
				}
			}
			if !anyMsgMatches {
				r.log.Debugf("Ignoring as any event message from %q doesn't match constraints %+v", strings.Join(event.Messages, ";"), rt.event.Message)
				return false, nil
			}
		}

		// resource name
		if rt.resourceName.AreConstraintsDefined() {
			allowed, err := rt.resourceName.IsAllowed(event.Name)
			if err != nil {
				return false, err
			}
			if !allowed {
				r.log.Debugf("Ignoring as resource name %q doesn't match constraints %+v", event.Name, rt.resourceName)
				return false, nil
			}
		}

		// namespace
		if rt.namespaces.AreConstraintsDefined() {
			match, err := rt.namespaces.IsAllowed(event.Namespace)
			if err != nil {
				return false, err
			}
			if !match {
				r.log.Debugf("Ignoring as namespace %q doesn't match constraints %+v", event.Namespace, rt.namespaces)
				return false, nil
			}
		}

		// annotations
		if !kvsSatisfiedForMap(rt.annotations, event.ObjectMeta.Annotations) {
			continue
		}

		// labels
		if !kvsSatisfiedForMap(rt.labels, event.ObjectMeta.Labels) {
			continue
		}
		return true, nil
	}

	return false, errs.ErrorOrNil()
}

func kvsSatisfiedForMap(expectedKV *map[string]string, obj map[string]string) bool {
	if expectedKV == nil || len(*expectedKV) == 0 {
		return true
	}

	if len(obj) == 0 {
		return false
	}

	for k, v := range *expectedKV {
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

func (r registration) qualifyEvent(
	event event.Event,
	newObj, oldObj interface{},
	routes []route,
) (bool, []string, error) {
	ok, err := r.matchEvent(routes, event)
	if err != nil {
		return false, nil, fmt.Errorf("while matching event: %w", err)
	}
	if !ok {
		return false, nil, nil
	}

	if event.Type == config.UpdateEvent {
		return r.qualifyEventForUpdate(newObj, oldObj, routes)
	}

	return true, nil, nil
}

func (r registration) qualifyEventForUpdate(
	newObj, oldObj interface{},
	routes []route,
) (bool, []string, error) {
	var diffs []string

	var oldUnstruct, newUnstruct *unstructured.Unstructured
	var ok bool

	if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
		r.log.Error("Failed to typecast old object to Unstructured.")
	}

	if newUnstruct, ok = newObj.(*unstructured.Unstructured); !ok {
		r.log.Error("Failed to typecast new object to Unstructured.")
	}

	for _, route := range routes {
		if !route.hasActionableUpdateSetting() {
			r.log.Debugf("Qualified for update: route: %v, with no updateSettings set", route)
			break
		}

		diff, err := k8sutil.Diff(oldUnstruct.Object, newUnstruct.Object, *route.updateSetting)
		if err != nil {
			r.log.Errorf("while getting diff: %w", err)
		}
		r.log.Debugf("About to qualify event for route: %v for update, diff: %s, updateSetting: %+v", route, diff, route.updateSetting)

		if len(diff) > 0 && route.updateSetting.IncludeDiff {
			diffs = append(diffs, diff)
			r.log.Debugf("Qualified for update: route: %v for update, diff: %s, updateSetting: %+v", route, diff, route.updateSetting)
		}
	}

	return true, diffs, nil
}

// gvrToString converts GVR formats to string.
func gvrToString(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}
