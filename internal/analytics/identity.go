package analytics

import (
	k8sVersion "k8s.io/apimachinery/pkg/version"

	"github.com/kubeshop/botkube/pkg/version"
)

// Identity defines an anonymous identity for a given installation.
type Identity struct {
	DeploymentID          string
	AnonymousID           string
	KubernetesVersion     k8sVersion.Info
	BotkubeVersion        version.Details
	WorkerNodeCount       int
	ControlPlaneNodeCount int
}

// TraitsMap returns a map with traits based on Identity struct fields.
func (i Identity) TraitsMap() map[string]interface{} {
	traits := map[string]interface{}{
		"k8sVersion":            i.KubernetesVersion,
		"botkubeVersion":        i.BotkubeVersion,
		"workerNodeCount":       i.WorkerNodeCount,
		"controlPlaneNodeCount": i.ControlPlaneNodeCount,
		"deploymentID":          i.DeploymentID,
	}

	if i.DeploymentID != "" {
		traits["deploymentID"] = i.DeploymentID
	}

	return traits
}
