package analytics

import (
	"github.com/kubeshop/botkube/pkg/version"
	k8sVersion "k8s.io/apimachinery/pkg/version"
)

// Identity defines an anonymous identity for a given installation.
type Identity struct {
	Cluster      ClusterIdentity
	Installation InstallationIdentity
}

// ClusterIdentity defines an anonymous cluster identity.
type ClusterIdentity struct {
	ID                string
	KubernetesVersion k8sVersion.Info
}

// TraitsMap returns a map with traits based on ClusterIdentity struct fields.
func (i ClusterIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"k8sVersion": i.KubernetesVersion,
	}
}

// InstallationIdentity defines an anonymous installation identity.
type InstallationIdentity struct {
	ID             string
	BotKubeVersion version.Details
}

// TraitsMap returns a map with traits based on InstallationIdentity struct fields.
func (i InstallationIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"botkubeVersion": i.BotKubeVersion,
	}
}
