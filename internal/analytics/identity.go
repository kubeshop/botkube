package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/kubeshop/botkube/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sVersion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
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

type Config struct {
	InstallationID string `json:"installationID"`
}

func NewConfig() Config {
	return Config{
		InstallationID: uuid.NewString(),
	}
}

const (
	kubeSystemNSName  = "kube-system"
	analyticsFileName = "analytics.yaml"
)

func CurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, cfgDir string) (Identity, error) {
	k8sServerVersion, err := k8sCli.Discovery().ServerVersion()
	if err != nil {
		return Identity{}, fmt.Errorf("while getting K8s server version: %w", err)
	}
	if k8sServerVersion == nil {
		return Identity{}, errors.New("server version object cannot be nil")
	}

	clusterID, err := getClusterID(ctx, k8sCli)
	if err != nil {
		return Identity{}, fmt.Errorf("while getting cluster ID: %w", err)
	}

	installationID, err := getInstallationID(cfgDir)
	if err != nil {
		return Identity{}, fmt.Errorf("while getting installation ID: %w", err)
	}

	return Identity{
		Cluster: ClusterIdentity{
			ID:                clusterID,
			KubernetesVersion: *k8sServerVersion,
		},
		Installation: InstallationIdentity{
			ID:             installationID,
			BotKubeVersion: version.Info(),
		},
	}, nil
}

func getClusterID(ctx context.Context, k8sCli kubernetes.Interface) (string, error) {
	kubeSystemNS, err := k8sCli.CoreV1().Namespaces().Get(ctx, kubeSystemNSName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("while getting %q Namespace: %w", kubeSystemNS, err)
	}
	if kubeSystemNS == nil {
		return "", errors.New("namespace object cannot be nil")
	}

	return string(kubeSystemNS.GetUID()), nil
}

func getInstallationID(cfgDir string) (string, error) {
	analyticsCfgFilePath := filepath.Join(cfgDir, analyticsFileName)
	if _, err := os.Stat(analyticsCfgFilePath); os.IsNotExist(err) {
		analyticsCfg := NewConfig()
		err = saveAnalyticsCfg(analyticsCfgFilePath, analyticsCfg)
		if err != nil {
			return "", err
		}

		return analyticsCfg.InstallationID, nil
	}

	analyticsCfgBytes, err := os.ReadFile(filepath.Clean(analyticsCfgFilePath))
	if err != nil {
		return "", fmt.Errorf("while reading analytics config file: %w", err)
	}

	var analyticsCfg Config
	err = json.Unmarshal(analyticsCfgBytes, &analyticsCfg)
	if err != nil {
		return "", fmt.Errorf("while unmarshalling analytics config file: %w", err)
	}

	if analyticsCfg.InstallationID == "" {
		analyticsCfg := NewConfig()
		err = saveAnalyticsCfg(analyticsCfgFilePath, analyticsCfg)
		if err != nil {
			return "", err
		}
		return analyticsCfg.InstallationID, nil
	}

	return analyticsCfg.InstallationID, nil
}

func saveAnalyticsCfg(path string, cfg Config) error {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("while marshalling analytics config file: %w", err)
	}

	err = os.WriteFile(path, bytes, 0600)
	if err != nil {
		return fmt.Errorf("while saving analytics config file to %q: %w", path, err)
	}

	return nil
}
