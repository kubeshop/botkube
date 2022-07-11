package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Reporter defines an analytics reporter implementation.
type Reporter interface {

	// RegisterCurrentIdentity loads the current anonymous identity and registers it.
	RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, cfgDir string) error
}
