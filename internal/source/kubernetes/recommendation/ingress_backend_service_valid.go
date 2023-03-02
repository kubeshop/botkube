package recommendation

import (
	"context"
	"fmt"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/k8sutil"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const ingressBackendServiceValidName = "IngressBackendServiceValid"

// IngressBackendServiceValid checks if service and tls secret used in ingress specs is already present
// and adds recommendations to event struct accordingly
type IngressBackendServiceValid struct {
	dynamicCli dynamic.Interface
}

// NewIngressBackendServiceValid creates a new IngressBackendServiceValid instance.
func NewIngressBackendServiceValid(dynamicCli dynamic.Interface) *IngressBackendServiceValid {
	return &IngressBackendServiceValid{dynamicCli: dynamicCli}
}

// Do executes the recommendation checks.
func (f *IngressBackendServiceValid) Do(ctx context.Context, event event.Event) (Result, error) {
	if event.Kind != "Ingress" || event.Type != config.CreateEvent || k8sutil.GetObjectTypeMetaData(event.Object).Kind == "Event" {
		return Result{}, nil
	}

	unstrObj, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		return Result{}, fmt.Errorf("cannot convert %T into type %T", event.Object, unstrObj)
	}

	var ingress networkingv1.Ingress
	err := k8sutil.TransformIntoTypedObject(unstrObj, &ingress)
	if err != nil {
		return Result{}, fmt.Errorf("while transforming object type %T into type: %T: %w", event.Object, ingress, err)
	}

	var warningMsgs []string
	errs := multierror.New()

	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if path.Backend.Service == nil {
				// TODO: Support path.Backend.Resource as well
				continue
			}
			serviceName := path.Backend.Service.Name
			servicePort := path.Backend.Service.Port.Number
			ns := f.findNamespaceFromService(serviceName, ingress.ObjectMeta.Namespace)

			svc, exists, err := f.getService(ctx, f.dynamicCli, serviceName, ns)
			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("while getting service: %w", err))
			}
			if !exists {
				warningMsgs = append(warningMsgs, fmt.Sprintf("Service '%s' referred in Ingress '%s/%s' spec does not exist.", serviceName, ingress.Namespace, ingress.Name))
				continue
			}

			valid := f.validateServicePort(svc, servicePort)
			if !valid {
				warningMsgs = append(warningMsgs, fmt.Sprintf("Service '%s' referred in Ingress '%s/%s' spec does not expose port '%d'.", serviceName, ingress.Namespace, ingress.Name, servicePort))
			}
		}
	}

	return Result{Warnings: warningMsgs}, errs.ErrorOrNil()
}

func (f *IngressBackendServiceValid) getService(ctx context.Context, dynamicCli dynamic.Interface, name, namespace string) (coreV1.Service, bool, error) {
	serviceGVR := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "services",
	}
	unstructuredService, err := dynamicCli.Resource(serviceGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return coreV1.Service{}, false, nil
		}
		return coreV1.Service{}, false, err
	}
	var svc coreV1.Service
	err = k8sutil.TransformIntoTypedObject(unstructuredService, &svc)
	if err != nil {
		return coreV1.Service{}, false, err
	}

	return svc, true, nil
}

func (f *IngressBackendServiceValid) validateServicePort(svc coreV1.Service, port int32) bool {
	for _, p := range svc.Spec.Ports {
		if p.Port == port {
			return true
		}
	}
	return false
}

func (f *IngressBackendServiceValid) findNamespaceFromService(service, defaultNS string) string {
	ns := strings.Split(service, ".")
	if len(ns) > 1 {
		return ns[1]
	}
	return defaultNS
}

// Name returns the recommendation name
func (f *IngressBackendServiceValid) Name() string {
	return ingressBackendServiceValidName
}
