package builder

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/authorization/v1"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/ptr"
)

// resourceVerbs is a list of verbs that are supported by Kubernetes api-server natively.
// Copied from: https://github.com/kubernetes/kubernetes/blob/release-1.29/staging/src/k8s.io/kubectl/pkg/cmd/auth/cani.go#L106
var resourceVerbs = []string{"get", "list", "watch", "create", "update", "patch", "delete", "deletecollection", "use", "bind", "impersonate", "*"}

// K8sAuth provides functionality to check if we have enough permissions to run given kubectl command.
type K8sAuth struct {
	cli v1.AuthorizationV1Interface
	log logrus.FieldLogger
}

// NewK8sAuth return a new K8sAuth instance.
func NewK8sAuth(cli v1.AuthorizationV1Interface) *K8sAuth {
	return &K8sAuth{
		cli: cli,
	}
}

// ValidateUserAccess validates that a given verbs are allowed. Returns user-facing message if not allowed.
func (c *K8sAuth) ValidateUserAccess(ns, verb, resource, name string) (bool, *api.Section) {
	var subresource string

	// kubectl logs/pods [NAME] should be translated into 'get logs pod [NAME]'
	// as the `log` is a subresource, same as scale, etc.
	//
	// We try to support as much as we can but this can grow even with custom plugins,
	// so we return warnings in case of unknown verbs instead of blocking the operation.
	switch verb {
	case "logs", "log":
		verb = "get"
		subresource = "log"
	case "describe":
		verb = "get"
	case "api-resources", "api-versions":
		// no specific permission needed
		return true, nil
	case "top":
		verb = "get"
		subresource = "metrics.k8s.io"
	}

	ctx := context.Background()
	review := authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace:   ns,
				Verb:        verb,
				Resource:    resource,
				Subresource: subresource,
				Name:        name,
			},
		},
	}
	out, err := c.cli.SelfSubjectAccessReviews().Create(ctx, &review, metav1.CreateOptions{})
	if err != nil {
		c.log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to create access review")
		return false, ptr.FromType(InternalErrorSection())
	}

	wasKnownVerb := slices.Contains(resourceVerbs, verb)

	if !wasKnownVerb && !out.Status.Allowed {
		// in this case we allow it anyway, as the API server may be wrong about it.
		return true, &api.Section{
			Context: []api.ContextItem{
				{
					Text: ":warning: Unable to verify if the action can be performed. Running this command may lead to an unauthorized error. To learn more about `kubectl` RBAC visit https://docs.botkube.io/configuration/executor/kubectl.",
				},
			},
		}
	}

	if !out.Status.Allowed {
		msg := ":exclamation: You don't have enough permission to run this command.\n"
		if out.Status.Reason != "" {
			msg = fmt.Sprintf("%sReason: %s\n", msg, out.Status.Reason)
		}
		return false, c.notAllowedMessage(msg)
	}

	return false, nil
}

func (c *K8sAuth) notAllowedMessage(msg string) *api.Section {
	return &api.Section{
		Base: api.Base{
			Header: "Missing permissions",
			Body: api.Body{
				Plaintext: msg,
			},
		},
		Context: []api.ContextItem{
			{
				Text: "To learn more about `kubectl` RBAC visit https://docs.botkube.io/configuration/executor/kubectl.",
			},
		},
	}
}
