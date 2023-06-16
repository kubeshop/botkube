package builder

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/command"
)

type (
	// NamespaceLister provides an option to list all namespaces in a given cluster.
	NamespaceLister interface {
		List(ctx context.Context, opts metav1.ListOptions) (*corev1.NamespaceList, error)
	}

	// KubectlRunner provides an option to run a given kubectl command.
	KubectlRunner interface {
		RunKubectlCommand(ctx context.Context, defaultNamespace, cmd string) (string, error)
	}

	// CommandGuard is an interface that allows to check if a given command is allowed to be executed.
	CommandGuard interface {
		GetAllowedResourcesForVerb(verb string, allConfiguredResources []string) ([]command.Resource, error)
		GetResourceDetails(verb, resourceType string) (command.Resource, error)
		FilterSupportedVerbs(allVerbs []string) []string
	}

	// AuthChecker provides an option to check if we can run a kubectl commands with a given permission.
	AuthChecker interface {
		CheckUserAccess(ns, verb, resource, name string) error
	}
)
