package recommendation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/internal/source/kubernetes/event"
	"github.com/kubeshop/botkube/internal/source/kubernetes/recommendation"
)

func TestIngressBackendServiceValid_Do_HappyPath(t *testing.T) {
	// given
	expected := recommendation.Result{
		Warnings: []string{
			"Service 'test-service' referred in Ingress 'default/ingress-with-service' spec does not exist.",
			"Service 'existing-service' referred in Ingress 'default/ingress-with-service' spec does not expose port '4000'.",
		},
	}

	dynamicCli := fake.NewSimpleDynamicClient(scheme.Scheme, fixService())
	recomm := recommendation.NewIngressBackendServiceValid(dynamicCli)

	ingress := fixIngressWithBackends()
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ingress)
	require.NoError(t, err)
	unstr := &unstructured.Unstructured{Object: unstrObj}

	event, err := event.New(ingress.ObjectMeta, unstr, config.CreateEvent, "networking.k8s.io/v1/ingresses")
	require.NoError(t, err)

	// when
	actual, err := recomm.Do(context.Background(), event)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func fixIngressWithBackends() *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-with-service",
			Namespace: "default",
		},
		Spec: networkingv1.IngressSpec{
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
												Number: 80,
											},
										},
									},
								},
								{
									Path: "testpath2",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "existing-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 4000,
											},
										},
									},
								},
								{
									Path: "testpath3",
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "existing-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 3001,
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

func fixService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-service",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{Port: 3000}, {Port: 3001}},
		},
	}
}
