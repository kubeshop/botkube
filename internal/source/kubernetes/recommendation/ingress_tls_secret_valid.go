package recommendation

import (
	"context"
	"fmt"

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

const ingressTLSSecretValidName = "IngressTLSSecretValid"

// IngressTLSSecretValid adds recommendations if TLS secrets used in Ingress specs don't exist.
type IngressTLSSecretValid struct {
	dynamicCli dynamic.Interface
}

// NewIngressTLSSecretValid creates a new IngressTLSSecretValid instance.
func NewIngressTLSSecretValid(dynamicCli dynamic.Interface) *IngressTLSSecretValid {
	return &IngressTLSSecretValid{dynamicCli: dynamicCli}
}

// Do executes the recommendation checks.
func (f *IngressTLSSecretValid) Do(ctx context.Context, event event.Event) (Result, error) {
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

	for _, tls := range ingress.Spec.TLS {
		exists, err := f.validateSecretExists(ctx, f.dynamicCli, tls.SecretName, ingress.Namespace)
		if err != nil {
			warningMsgs = append(warningMsgs, fmt.Sprintf("TLS secret '%s' does not exist", tls.SecretName))
		}

		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while validating TLS secret existence: %w", err))
		}

		if !exists {
			warningMsgs = append(warningMsgs, fmt.Sprintf("TLS secret '%s' referred in Ingress '%s/%s' does not exist.", tls.SecretName, ingress.Namespace, ingress.Name))
		}
	}

	return Result{Warnings: warningMsgs}, errs.ErrorOrNil()
}

func (f *IngressTLSSecretValid) validateSecretExists(ctx context.Context, dynamicCli dynamic.Interface, name, namespace string) (bool, error) {
	secretGVR := schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}
	_, err := dynamicCli.Resource(secretGVR).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// Name returns the recommendation name
func (f *IngressTLSSecretValid) Name() string {
	return ingressTLSSecretValidName
}
