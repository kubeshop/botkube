package accessreview

import (
	"context"
	"errors"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

// K8sAuth provides functionality to check if we have enough permissions to run given kubectl command.
type K8sAuth struct {
	cli v1.AuthorizationV1Interface
}

// NewK8sAuth return a new K8sAuth instance.
func NewK8sAuth(cli v1.AuthorizationV1Interface) *K8sAuth {
	return &K8sAuth{
		cli: cli,
	}
}

// CheckUserAccess returns error if a given verbs are not supported.
func (c *K8sAuth) CheckUserAccess(ns, verb, resource, name string) error {
	var subresource string

	// kubectl logs/pods [NAME] should be translated into 'get logs pod [NAME]'
	// as the `log` is a subresource, same as scale, etc.
	//
	// TODO: only logs are supported by interactive builder. We don't support scale, exec, apply, etc.
	// Once we will add support for them, we need to add dedicated cases here.
	switch verb {
	case "logs", "log":
		verb = "get"
		subresource = "log"
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
		return fmt.Errorf("while creating access review: %w", err)
	}

	if !out.Status.Allowed {
		msg := ":exclamation: You don't have enough permission to run this command.\n"
		if out.Status.Reason != "" {
			msg = fmt.Sprintf("%sReason: %s\n", msg, out.Status.Reason)
		}
		return errors.New(msg)
	}

	return nil
}
