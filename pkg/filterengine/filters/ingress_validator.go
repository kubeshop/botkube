package filters

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/utils"
)

// IngressValidator checks if service and tls secret used in ingress specs is already present
// and adds recommendations to event struct accordingly
type IngressValidator struct {
	log        logrus.FieldLogger
	dynamicCli dynamic.Interface
}

// NewIngressValidator creates a new IngressValidator instance
func NewIngressValidator(log logrus.FieldLogger, dynamicCli dynamic.Interface) *IngressValidator {
	return &IngressValidator{log: log, dynamicCli: dynamicCli}
}

// Run filers and modifies event struct
func (f *IngressValidator) Run(ctx context.Context, object interface{}, event *events.Event) error {
	if event.Kind != "Ingress" || event.Type != config.CreateEvent || utils.GetObjectTypeMetaData(object).Kind == "Event" {
		return nil
	}
	var ingressObj networkingv1.Ingress
	err := utils.TransformIntoTypedObject(object.(*unstructured.Unstructured), &ingressObj)
	if err != nil {
		return fmt.Errorf("while transforming object type %T into type: %T: %w", object, ingressObj, err)
	}

	ingNs := ingressObj.ObjectMeta.Namespace

	// Check if service names are valid in the configuration
	for _, rule := range ingressObj.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			if path.Backend.Service == nil {
				// TODO: Support path.Backend.Resource as well
				continue
			}
			serviceName := path.Backend.Service.Name
			servicePort := path.Backend.Service.Port.Number
			ns := FindNamespaceFromService(serviceName)
			if ns == "default" {
				ns = ingNs
			}
			_, err := ValidServicePort(ctx, f.dynamicCli, serviceName, ns, servicePort)
			if err != nil {
				event.Warnings = append(event.Warnings, fmt.Sprintf("Service '%s' used in ingress '%s' config does not exist or port '%v' not exposed", serviceName, ingressObj.Name, servicePort))
			}
		}
	}

	// Check if tls secret exists
	for _, tls := range ingressObj.Spec.TLS {
		_, err := ValidSecret(ctx, f.dynamicCli, tls.SecretName, ingNs)
		if err != nil {
			event.Warnings = append(event.Warnings, fmt.Sprintf("TLS secret '%s' does not exist", tls.SecretName))
		}
	}
	f.log.Debug("Ingress Validator filter successful!")
	return nil
}

// Name returns the filter's name
func (f *IngressValidator) Name() string {
	return "IngressValidator"
}

// Describe describes the filter
func (f *IngressValidator) Describe() string {
	return "Checks if services and tls secrets used in ingress specs are available."
}
