package sources

import (
	"context"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/utils"
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

func (r registration) handleEvent(ctx context.Context, resource string, target config.EventType, sourceRoutes []route, fn eventHandler) {
	switch target {
	case config.CreateEvent:
		r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				sources, err := sourcesForObj(ctx, sourceRoutes, obj, r.log, r.mapper, r.dynamicCli)
				if err != nil {
					r.log.WithFields(logrus.Fields{
						"eventHandler": config.CreateEvent,
						"resource":     resource,
						"error":        err.Error(),
					}).Errorf("Cannot calculate sources for observed resource.")
					return
				}
				r.log.Debugf("handle Create event, resource: %q, sources: %+v", resource, sources)
				if len(sources) > 0 {
					fn(ctx, resource, sources, nil)(obj)
				}
			},
		})
	case config.DeleteEvent:
		r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				sources, err := sourcesForObj(ctx, sourceRoutes, obj, r.log, r.mapper, r.dynamicCli)
				if err != nil {
					r.log.WithFields(logrus.Fields{
						"eventHandler": config.DeleteEvent,
						"resource":     resource,
						"error":        err.Error(),
					}).Errorf("Cannot calculate sources for observed resource.")
					return
				}
				r.log.Debugf("handle Delete event, resource: %q, sources: %+v", resource, sources)
				if len(sources) > 0 {
					fn(ctx, resource, sources, nil)(obj)
				}
			},
		})
	case config.UpdateEvent:
		r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				sources, diffs, err := qualifySourcesForUpdate(ctx, newObj, oldObj, sourceRoutes, r.log, r.mapper, r.dynamicCli)
				if err != nil {
					r.log.WithFields(logrus.Fields{
						"eventHandler": config.UpdateEvent,
						"resource":     resource,
						"error":        err.Error(),
					}).Errorf("Cannot qualify sources for observed resource.")
					return
				}
				r.log.Debugf("handle Update event, resource: %s, sources: %+v, diffs: %+v", resource, sources, diffs)
				if len(sources) > 0 {
					fn(ctx, resource, sources, diffs)(newObj)
				}
			},
		})
	}
}

func (r registration) handleMapped(ctx context.Context, targetEvent config.EventType, routeTable map[string][]entry, fn eventHandler) {
	r.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var eventObj coreV1.Event
			err := utils.TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
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
			gvr, err := utils.GetResourceFromKind(r.mapper, eventObj.InvolvedObject.GroupVersionKind())
			if err != nil {
				r.log.Errorf("Failed to get involved object: %v", err)
				return
			}

			if !r.canHandleEvent(eventObj.Type) {
				return
			}

			gvrToString := utils.GVRToString(gvr)
			if !r.includesSrcResource(gvrToString) {
				return
			}

			sourceRoutes := sourceRoutes(routeTable, gvrToString, targetEvent)
			sources, err := sourcesForObj(ctx, sourceRoutes, obj, r.log, r.mapper, r.dynamicCli)
			if err != nil {
				r.log.Errorf("cannot calculate sources for observed mapped resource event: %q in Add event handler: %s", targetEvent, err.Error())
				return
			}
			if len(sources) == 0 {
				return
			}
			fn(ctx, gvrToString, sources, nil)(obj)
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

func sourcesForObj(ctx context.Context, routes []route, obj interface{}, log logrus.FieldLogger, mapper meta.RESTMapper, cli dynamic.Interface) ([]string, error) {
	var out []string

	objectMeta, err := utils.GetObjectMetaData(ctx, cli, mapper, obj)
	if err != nil {
		log.Errorf("while getting object metadata: %s", err.Error())
		return nil, err
	}

	// resource name


	// annotations

	// labels

	// namespace
	targetNs := objectMeta.Namespace
	if targetNs == "" {
		log.Debugf("handling event for cluster-wide resource in routes: %+v", targetNs, routes)
		for _, route := range routes {
			out = append(out, route.source)
		}

		return out, nil
	}

	log.Debugf("handling events for target Namespace: %s in routes: %+v", targetNs, routes)
	for _, route := range routes {
		if route.namespaces.IsAllowed(targetNs) {
			out = append(out, route.source)
		}
	}

	return out, nil
}

func qualifySourcesForUpdate(
	ctx context.Context,
	newObj, oldObj interface{},
	routes []route,
	log logrus.FieldLogger,
	mapper meta.RESTMapper,
	cli dynamic.Interface,
) ([]string, []string, error) {
	var sources, diffs []string

	candidates, err := sourcesForObj(ctx, routes, newObj, log, mapper, cli)
	if err != nil {
		return nil, nil, err
	}

	var oldUnstruct, newUnstruct *unstructured.Unstructured
	var ok bool

	if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
		log.Error("Failed to typecast old object to Unstructured.")
	}

	if newUnstruct, ok = newObj.(*unstructured.Unstructured); !ok {
		log.Error("Failed to typecast new object to Unstructured.")
	}

	log.Debugf("qualifySourcesForUpdate source candidates: %+v", candidates)

	for _, source := range candidates {
		for _, r := range routes {
			if r.source != source {
				continue
			}

			if !r.hasActionableUpdateSetting() {
				log.Debugf("Qualified for update: source: %s, with no updateSettings set", source)
				sources = append(sources, source)
				continue
			}

			diff, err := utils.Diff(oldUnstruct.Object, newUnstruct.Object, r.updateSetting)
			if err != nil {
				log.Errorf("while getting diff: %w", err)
			}
			log.Debugf("About to qualify source: %s for update, diff: %s, updateSetting: %+v", source, diff, r.updateSetting)

			if len(diff) > 0 && r.updateSetting.IncludeDiff {
				sources = append(sources, source)
				diffs = append(diffs, diff)
				log.Debugf("Qualified for update: source: %s for update, diff: %s, updateSetting: %+v", source, diff, r.updateSetting)
			}
		}
	}

	return sources, diffs, nil
}
