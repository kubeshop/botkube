package main

import (
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine/utils"
	extV1beta1 "k8s.io/api/extensions/v1beta1"
)

// IngressValidator checks if service and tls secret used in ingress specs is already present
// and adds recommendations to event struct accordingly
type IngressValidator struct {
}

// Create new IngressValidator
var Filter IngressValidator

// Run filers and modifies event struct
func (iv *IngressValidator) Run(object interface{}, event *events.Event) {
	if event.Kind != "Ingress" && event.Type != "create" {
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
			ns := utils.FindNamespaceFromService(serviceName)
			if ns == "default" {
				ns = ingNs
			}
			_, err := utils.ValidServicePort(serviceName, ns, int32(servicePort))
			if err != nil {
				event.Messages = append(event.Messages, "Service "+serviceName+" used in ingress config does not exist or port not exposed\n")
				event.Level = events.Warn
			}
		}

	}

	// Check if tls secret exists
	for _, tls := range ingressObj.Spec.TLS {
		_, err := utils.ValidSecret(tls.SecretName, ingNs)
		if err != nil {
			event.Recommendations = append(event.Recommendations, "TLS secret "+tls.SecretName+"does not exist")
		}
	}
}
