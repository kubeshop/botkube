package filters

import (
	"context"
	"testing"

	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
)

func TestIngressValidator_Run_HappyPath(t *testing.T) {
	// given
	dynamicCli := fake.NewSimpleDynamicClient(runtime.NewScheme())
	logger, _ := logtest.NewNullLogger()
	ingressValidator := NewIngressValidator(logger, dynamicCli)

	ingress := fixIngress()
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ingress)
	require.NoError(t, err)
	unstr := &unstructured.Unstructured{Object: unstrObj}

	event, err := events.New(ingress.ObjectMeta, unstr, config.CreateEvent, "networking.k8s.io/v1/ingresses", "sample")
	require.NoError(t, err)

	// when
	err = ingressValidator.Run(context.Background(), unstr, &event)

	// then
	assert.NoError(t, err)

	require.Len(t, event.Warnings, 2)
	assert.Contains(t, event.Warnings, "Service 'test-service' used in ingress 'ingress-with-service' config does not exist or port '80' not exposed")
	assert.Contains(t, event.Warnings, "TLS secret 'not-existing' does not exist")
}

func fixIngress() *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ingress-with-service",
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{Hosts: []string{"foo"}, SecretName: "not-existing"},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "foo",
					IngressRuleValue: networkingv1.IngressRuleValue{

						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "testpath",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: int32(80),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
