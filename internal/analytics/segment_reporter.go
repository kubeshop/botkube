package analytics

import (
	"context"
	"errors"
	"fmt"

	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/version"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
)

const (
	kubeSystemNSName  = "kube-system"
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
func (r *SegmentReporter) RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface) error {
	currentIdentity, err := r.load(ctx, k8sCli)
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
func (r *SegmentReporter) ReportCommand(platform config.CommPlatformIntegration, command string, origin command.Origin, withFilter bool) error {
	return r.reportEvent("Command executed", map[string]interface{}{
		"platform": platform,
		"command":  command,
		"origin":   origin,
		"filtered": withFilter,
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
		anonymousID = r.identity.ID
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
		AnonymousId: r.identity.ID,
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
		AnonymousId: identity.ID,
		Traits:      identity.TraitsMap(),
	})
	if err != nil {
		return fmt.Errorf("while enqueuing identify message: %w", err)
	}

	r.identity = &identity
	return nil
}

func (r *SegmentReporter) load(ctx context.Context, k8sCli kubernetes.Interface) (Identity, error) {
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

	workerNodeCount, controlPlaneNodeCount, err := r.getNodeCount(ctx, k8sCli)
	if err != nil {
		return Identity{}, fmt.Errorf("while getting node count: %w", err)
	}

	return Identity{
		ID:                    clusterID,
		KubernetesVersion:     *k8sServerVersion,
		BotkubeVersion:        version.Info(),
		WorkerNodeCount:       workerNodeCount,
		ControlPlaneNodeCount: controlPlaneNodeCount,
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

func (r *SegmentReporter) getNodeCount(ctx context.Context, k8sCli kubernetes.Interface) (int, int, error) {
	nodeList, err := k8sCli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err != nil {
		return 0, 0, fmt.Errorf("while getting node count: %w", err)
	}

	var (
		controlPlaneNodesCount int
		workerNodesCount       int
	)

	for _, item := range nodeList.Items {
		val, ok := item.Labels[kubeadmconstants.LabelNodeRoleControlPlane]
		if !ok || val != "true" {
			workerNodesCount++
			continue
		}
		controlPlaneNodesCount++
	}

	return workerNodesCount, controlPlaneNodesCount, nil
}
