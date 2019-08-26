package filters

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
)

// IngressValidator checks if service and tls secret used in ingress specs is already present
// and adds recommendations to event struct accordingly
type IngressValidator struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(IngressValidator{
		Description: "Checks if services and tls secrets used in ingress specs are available.",
	})
}

// Run filers and modifies event struct
func (iv IngressValidator) Run(object interface{}, event *events.Event) {
	if event.Kind != "Ingress" || event.Type != config.CreateEvent {
		return
	}
	ingressObj, ok := object.(*extV1beta1.Ingress)
	if !ok {
		return
	}

	ingNs := ingressObj.ObjectMeta.Namespace

	// Check if service names are valid in the configuration
	for _, rule := range ingressObj.Spec.Rules {
		for _, path := range rule.IngressRuleValue.HTTP.Paths {
			serviceName := path.Backend.ServiceName
			servicePort := path.Backend.ServicePort.IntValue()
			ns := FindNamespaceFromService(serviceName)
			if ns == "default" {
				ns = ingNs
			}
			_, err := ValidServicePort(serviceName, ns, int32(servicePort))
			if err != nil {
				event.Warnings = append(event.Warnings, fmt.Sprintf("Service '%s' used in ingress '%s' config does not exist or port '%v' not exposed\n", serviceName, ingressObj.Name, servicePort))
			}
		}
	}

	// Check if tls secret exists
	for _, tls := range ingressObj.Spec.TLS {
		_, err := ValidSecret(tls.SecretName, ingNs)
		if err != nil {
			event.Recommendations = append(event.Recommendations, "TLS secret "+tls.SecretName+"does not exist")
		}
	}
	log.Logger.Debug("Ingress Validator filter successful!")
}

// Describe filter
func (iv IngressValidator) Describe() string {
	return iv.Description
}
