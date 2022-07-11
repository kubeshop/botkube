package analytics

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

type Reporter interface {
	RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, cfgDir string) error
}
