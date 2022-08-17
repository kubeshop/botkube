package sources

import (
	"context"
	"reflect"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/utils"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

type Registration struct {
	informer        cache.SharedIndexInformer
	log             logrus.FieldLogger
	mapper          meta.RESTMapper
	dynamicCli      dynamic.Interface
	events          []config.EventType
	mappedResources []string
	mappedEvent     config.EventType
}

func (i Registration) handleEvent(ctx context.Context, resource string, target config.EventType, sourceRoutes []Route, fn eventHandler) {
	switch target {
	case config.CreateEvent:
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				sources := sourcesForObjNamespace(ctx, sourceRoutes, obj, i.log, i.mapper, i.dynamicCli)
				i.log.Debugf("handleEvent - CreateEvent - resource: %s, sources: %+v", resource, sources)
				if len(sources) > 0 {
					fn(ctx, resource, sources, nil)(obj)
				}
			},
		})
	case config.DeleteEvent:
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				sources := sourcesForObjNamespace(ctx, sourceRoutes, obj, i.log, i.mapper, i.dynamicCli)
				i.log.Debugf("handleEvent - DeleteEvent - resource: %s, sources: %+v", resource, sources)
				if len(sources) > 0 {
					fn(ctx, resource, sources, nil)(obj)
				}
			},
		})
	case config.UpdateEvent:
		i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				sources, diffs := qualifySourcesForUpdate(ctx, newObj, oldObj, sourceRoutes, i.log, i.mapper, i.dynamicCli)
				i.log.Debugf("handleEvent - UpdateEvent - resource: %s, sources: %+v, diffs: %+v", resource, sources, diffs)
				if len(sources) > 0 {
					fn(ctx, resource, sources, diffs)(newObj)
				}
			},
		})
	}
}

func qualifySourcesForUpdate(
	ctx context.Context,
	newObj, oldObj interface{},
	routes []Route,
	log logrus.FieldLogger,
	mapper meta.RESTMapper,
	cli dynamic.Interface,
) ([]string, []string) {
	var sources, diffs []string

	candidates := sourcesForObjNamespace(ctx, routes, newObj, log, mapper, cli)

	var oldUnstruct, newUnstruct *unstructured.Unstructured
	var ok bool

	if oldUnstruct, ok = oldObj.(*unstructured.Unstructured); !ok {
		log.Error("Failed to typecast object to Unstructured.")
	}

	if newUnstruct, ok = newObj.(*unstructured.Unstructured); !ok {
		log.Error("Failed to typecast object to Unstructured.")
	}

	log.Debugf("qualifySourcesForUpdate source candidates: %+v", qualifySourcesForUpdate)

	for _, source := range candidates {
		for _, r := range routes {
			if r.source != source || !r.hasActionableUpdateSetting() {
				continue
			}

			diff, err := utils.Diff(oldUnstruct.Object, newUnstruct.Object, r.updateSetting)
			if err != nil {
				log.Errorf("while getting diff: %w", err)
			}
			log.Debugf("qualifySourcesForUpdate source: %s, diff: %s, updateSetting: %+v", source, diff, r.updateSetting)

			if len(diff) > 0 && r.updateSetting.IncludeDiff {
				sources = append(sources, source)
				diffs = append(diffs, diff)
			}
		}
	}

	return sources, diffs
}

func (i Registration) handleMapped(ctx context.Context, targetEvent config.EventType, routeTable map[string][]Entry, fn eventHandler) {
	i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var eventObj coreV1.Event
			err := utils.TransformIntoTypedObject(obj.(*unstructured.Unstructured), &eventObj)
			if err != nil {
				i.log.Errorf("Unable to transform object type: %v, into type: %v", reflect.TypeOf(obj), reflect.TypeOf(eventObj))
			}
			_, err = cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				i.log.Errorf("Failed to get MetaNamespaceKey from event resource")
				return
			}

			// Find involved object type
			gvr, err := utils.GetResourceFromKind(i.mapper, eventObj.InvolvedObject.GroupVersionKind())
			if err != nil {
				i.log.Errorf("Failed to get involved object: %v", err)
				return
			}

			if !i.canHandleEvent(eventObj.Type) {
				return
			}

			gvrToString := utils.GVRToString(gvr)
			if !i.includesSrcResource(gvrToString) {
				return
			}

			sourceRoutes := sourceRoutes(routeTable, gvrToString, targetEvent)
			sources := sourcesForObjNamespace(ctx, sourceRoutes, obj, i.log, i.mapper, i.dynamicCli)
			if len(sources) == 0 {
				return
			}
			fn(ctx, gvrToString, sources, nil)(obj)
		},
	})
}

func (i Registration) canHandleEvent(target string) bool {
	for _, e := range i.events {
		if strings.ToLower(target) == e.String() {
			return true
		}
	}
	return false
}

func (i Registration) includesSrcResource(resource string) bool {
	for _, src := range i.mappedResources {
		if src == resource {
			return true
		}
	}
	return false
}

func sourcesForObjNamespace(ctx context.Context, routes []Route, obj interface{}, log logrus.FieldLogger, mapper meta.RESTMapper, cli dynamic.Interface) []string {
	var out []string

	objectMeta, err := utils.GetObjectMetaData(ctx, cli, mapper, obj)
	if err != nil {
		log.Errorf("while getting object metadata: %s", err.Error())
		return []string{}
	}

	targetNs := objectMeta.Namespace
	log.Debugf("handling events for target Namespace: %s in routes: %+v", targetNs, routes)

	for _, route := range routes {
		if route.namespaces.IsAllowed(targetNs) {
			out = append(out, route.source)
		}
	}

	return out
}
