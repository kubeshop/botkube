package analytics

var (
	// APIKey contains the API key for external analytics service. It is set during application build.
	APIKey string
)

type Reporter interface {
	RegisterIdentity(identity Identity) error
}

type CleanupFn func() error

type Identity struct {
	Cluster      ClusterIdentity
	Installation InstallationIdentity
}

type ClusterIdentity struct {
	ID                string
	KubernetesVersion string
}

func (i ClusterIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"k8sVersion": i.KubernetesVersion,
	}
}

type InstallationIdentity struct {
	ID             string
	BotKubeVersion string
	Notifiers      map[string]bool
	Bots           map[string]bool
}

func (i InstallationIdentity) TraitsMap() map[string]interface{} {
	return map[string]interface{}{
		"botkubeVersion": i.BotKubeVersion,
		"notifiers":      i.Notifiers,
		"bots":           i.Bots,
	}
}

func CurrentIdentity() (Identity, error) {
	anonymousID := ""
	clusterID := ""

}
