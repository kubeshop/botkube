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

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	kubeSystemNSName  = "kube-system"
	analyticsFileName = "analytics.yaml"
	unknownIdentityID = "00000000-0000-0000-0000-000000000000"
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

// NewSegmentReporter creates a new SegmentReporter instance.
func NewSegmentReporter(log logrus.FieldLogger, cli segment.Client) *SegmentReporter {
	return &SegmentReporter{
		log: log,
		cli: cli,
	}
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

// ReportCommand reports a new executed command. The command should be anonymized before using this method.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportCommand(platform config.CommPlatformIntegration, command string) error {
	return r.reportEvent("Command executed", map[string]interface{}{
		"platform": platform,
		"command":  command,
	})
}

// ReportBotEnabled reports an enabled bot.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportBotEnabled(platform config.CommPlatformIntegration) error {
	return r.reportEvent("Integration enabled", map[string]interface{}{
		"platform": platform,
		"type":     config.BotIntegrationType,
	})
}

// ReportSinkEnabled reports an enabled sink.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportSinkEnabled(platform config.CommPlatformIntegration) error {
	return r.reportEvent("Integration enabled", map[string]interface{}{
		"platform": platform,
		"type":     config.SinkIntegrationType,
	})
}

// ReportHandledEventSuccess reports a successfully handled event using a given communication platform.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportHandledEventSuccess(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails EventDetails) error {
	return r.reportEvent("Event handled", map[string]interface{}{
		"platform": platform,
		"type":     integrationType,
		"event":    eventDetails,
		"success":  true,
	})
}

// ReportHandledEventError reports a failure while handling event using a given communication platform.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportHandledEventError(integrationType config.IntegrationType, platform config.CommPlatformIntegration, eventDetails EventDetails, err error) error {
	return r.reportEvent("Event handled", map[string]interface{}{
		"platform": platform,
		"type":     integrationType,
		"event":    eventDetails,
		"error":    err.Error(),
	})
}

// ReportFatalError reports a fatal app error.
// It doesn't need a registered identity.
func (r *SegmentReporter) ReportFatalError(err error) error {
	properties := map[string]interface{}{
		"error": err.Error(),
	}

	var anonymousID string
	if r.identity != nil {
		anonymousID = r.identity.Installation.ID
	} else {
		anonymousID = unknownIdentityID
		properties["unknownIdentity"] = true
	}

	eventName := "Fatal error"
	err = r.cli.Enqueue(segment.Track{
		AnonymousId: anonymousID,
		Event:       eventName,
		Properties:  properties,
	})
	if err != nil {
		return fmt.Errorf("while enqueuing report of event %q: %w", eventName, err)
	}

	return nil
}

// Close cleans up the reporter resources.
func (r *SegmentReporter) Close() error {
	r.log.Info("Closing...")
	return r.cli.Close()
}

func (r *SegmentReporter) reportEvent(event string, properties map[string]interface{}) error {
	if r.identity == nil {
		return errors.New("identity needs to be registered first")
	}

	err := r.cli.Enqueue(segment.Track{
		AnonymousId: r.identity.Installation.ID,
		Event:       event,
		Properties:  properties,
	})
	if err != nil {
		return fmt.Errorf("while enqueuing report of event %q: %w", event, err)
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
		return "", fmt.Errorf("while getting %q Namespace: %w", kubeSystemNSName, err)
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
