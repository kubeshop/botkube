package analytics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/strings"

	"github.com/kubeshop/botkube/pkg/version"
)

const (
	kubeSystemNSName     = "kube-system"
	analyticsFileName    = "analytics.yaml"
	printAPIKeyCharCount = 3
)

var (
	// APIKey contains the API key for external analytics service. It is set during application build.
	APIKey string
)

var _ Reporter = &SegmentReporter{}

// SegmentReporter is a default Reporter implementation that uses Twilio Segment.
type SegmentReporter struct {
	log logrus.FieldLogger
	cli segment.Client

	identity *Identity
}

// CleanupFn defines a function which should be called to clean up SegmentReporter resources.
type CleanupFn func() error

// NewSegmentReporter creates a new SegmentReporter instance.
func NewSegmentReporter(log logrus.FieldLogger) (*SegmentReporter, CleanupFn, error) {
	log.Infof("Using API Key starting with %q...", strings.ShortenString(APIKey, printAPIKeyCharCount))
	cli, err := segment.NewWithConfig(APIKey, segment.Config{
		Logger:  newLoggerAdapter(log),
		Verbose: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("while creating new Analytics Client: %w", err)
	}

	cleanupFn := func() error {
		log.Info("Closing...")
		return cli.Close()
	}

	return &SegmentReporter{
			log: log,
			cli: cli,
		},
		cleanupFn,
		nil
}

// RegisterCurrentIdentity loads the current anonymous identity and registers it.
func (r *SegmentReporter) RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, cfgDir string) error {
	currentIdentity, err := r.load(ctx, k8sCli, cfgDir)
	if err != nil {
		return fmt.Errorf("while loading current identity: %w", err)
	}

	err = r.registerIdentity(currentIdentity)
	if err != nil {
		return fmt.Errorf("while registering identity: %w", err)
	}

	return nil
}

func (r *SegmentReporter) registerIdentity(identity Identity) error {
	err := r.cli.Enqueue(segment.Identify{
		AnonymousId: identity.Installation.ID,
		Traits:      identity.Installation.TraitsMap(),
	})
	if err != nil {
		return fmt.Errorf("while enqueuing identify message: %w", err)
	}

	err = r.cli.Enqueue(segment.Group{
		AnonymousId: identity.Installation.ID,
		GroupId:     identity.Cluster.ID,
		Traits:      identity.Cluster.TraitsMap(),
	})
	if err != nil {
		return fmt.Errorf("while enqueuing group message: %w", err)
	}

	r.identity = &identity
	return nil
}

func (r *SegmentReporter) load(ctx context.Context, k8sCli kubernetes.Interface, cfgDir string) (Identity, error) {
	k8sServerVersion, err := k8sCli.Discovery().ServerVersion()
	if err != nil {
		return Identity{}, fmt.Errorf("while getting K8s server version: %w", err)
	}
	if k8sServerVersion == nil {
		return Identity{}, errors.New("server version object cannot be nil")
	}

	clusterID, err := r.getClusterID(ctx, k8sCli)
	if err != nil {
		return Identity{}, fmt.Errorf("while getting cluster ID: %w", err)
	}

	installationID, err := r.getInstallationID(cfgDir)
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

func (r *SegmentReporter) getClusterID(ctx context.Context, k8sCli kubernetes.Interface) (string, error) {
	kubeSystemNS, err := k8sCli.CoreV1().Namespaces().Get(ctx, kubeSystemNSName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("while getting %q Namespace: %w", kubeSystemNS, err)
	}
	if kubeSystemNS == nil {
		return "", errors.New("namespace object cannot be nil")
	}

	return string(kubeSystemNS.GetUID()), nil
}

func (r *SegmentReporter) getInstallationID(cfgDir string) (string, error) {
	analyticsCfgFilePath := filepath.Join(cfgDir, analyticsFileName)
	if _, err := os.Stat(analyticsCfgFilePath); os.IsNotExist(err) {
		r.log.Info("Analytics configuration file is not found. Creating and saving one...")
		analyticsCfg := NewConfig()
		err = r.saveAnalyticsCfg(analyticsCfgFilePath, analyticsCfg)
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
	err = yaml.Unmarshal(analyticsCfgBytes, &analyticsCfg)
	if err != nil {
		return "", fmt.Errorf("while unmarshalling analytics config file: %w", err)
	}

	if analyticsCfg.InstallationID == "" {
		r.log.Info("Installation ID is empty. Generating one and saving...")
		analyticsCfg := NewConfig()
		err = r.saveAnalyticsCfg(analyticsCfgFilePath, analyticsCfg)
		if err != nil {
			return "", err
		}
		return analyticsCfg.InstallationID, nil
	}

	return analyticsCfg.InstallationID, nil
}

func (r *SegmentReporter) saveAnalyticsCfg(path string, cfg Config) error {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("while marshalling analytics config file: %w", err)
	}

	err = os.WriteFile(path, bytes, 0600)
	if err != nil {
		return fmt.Errorf("while saving analytics config file to %q: %w", path, err)
	}

	return nil
}
