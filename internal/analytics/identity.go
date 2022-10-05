package analytics

import (
	k8sVersion "k8s.io/apimachinery/pkg/version"

	"github.com/kubeshop/botkube/pkg/version"
)

// Identity defines an anonymous identity for a given installation.
type Identity struct {
	ID                string
	KubernetesVersion k8sVersion.Info
	BotkubeVersion    version.Details
}

// TraitsMap returns a map with traits based on Identity struct fields.
func (i Identity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"k8sVersion":     i.KubernetesVersion,
		"botkubeVersion": i.BotkubeVersion,
	}
}
