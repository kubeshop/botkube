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

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/recommendation"
)

func TestIngressTLSSecretValid_Do_HappyPath(t *testing.T) {
	// given
	expected := recommendation.Result{
		Warnings: []string{
			"TLS secret 'not-existing' referred in Ingress 'foo/ingress-with-service' does not exist.",
		},
	}

	dynamicCli := fake.NewSimpleDynamicClient(scheme.Scheme, fixSecret())
	recomm := recommendation.NewIngressTLSSecretValid(dynamicCli)

	ingress := fixIngressWithTLS()
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&ingress)
	require.NoError(t, err)
	unstr := &unstructured.Unstructured{Object: unstrObj}

	event, err := events.New(ingress.ObjectMeta, unstr, config.CreateEvent, "networking.k8s.io/v1/ingresses")
	require.NoError(t, err)

	// when
	actual, err := recomm.Do(context.Background(), event)

	// then
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func fixIngressWithTLS() *networkingv1.Ingress {
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-with-service",
			Namespace: "foo",
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{Hosts: []string{"foo"}, SecretName: "not-existing"},
				{Hosts: []string{"foo", "bar"}, SecretName: "existing-secret"},
			},
		},
	}
}

func fixSecret() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-secret",
			Namespace: "foo",
		},
	}
}
