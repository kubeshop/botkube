package builder

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

type (
	// NamespaceLister provides an option to list all namespaces in a given cluster.
	NamespaceLister interface {
		List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
	}

	KubectlRunner interface {
		RunKubectlCommand(ctx context.Context, defaultNamespace, cmd string) (string, error)
	}

	// CommandGuard is an interface that allows to check if a given command is allowed to be executed.
	CommandGuard interface {
		GetAllowedResourcesForVerb(verb string, allConfiguredResources []string) ([]kubectl.Resource, error)
		GetResourceDetails(verb, resourceType string) (kubectl.Resource, error)
		FilterSupportedVerbs(allVerbs []string) []string
	}
)
