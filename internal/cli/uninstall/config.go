package uninstall

import "github.com/kubeshop/botkube/internal/cli/uninstall/helm"

// Config holds parameters for Botkube deletion.
type Config struct {
	HelmParams helm.Config
}
