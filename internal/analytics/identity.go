package analytics

import (
	"github.com/kubeshop/botkube/pkg/version"
	k8sVersion "k8s.io/apimachinery/pkg/version"
)

type Identity struct {
	Cluster      ClusterIdentity
	Installation InstallationIdentity
}

type ClusterIdentity struct {
	ID                string
	KubernetesVersion k8sVersion.Info
}

func (i ClusterIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"k8sVersion": i.KubernetesVersion,
	}
}

type InstallationIdentity struct {
	ID             string
	BotKubeVersion version.Details
}

func (i InstallationIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"botkubeVersion": i.BotKubeVersion,
	}
}
