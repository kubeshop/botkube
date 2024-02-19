package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	segment "github.com/segmentio/analytics-go"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/internal/analytics/batched"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/plugin"
	"github.com/kubeshop/botkube/pkg/ptr"
	"github.com/kubeshop/botkube/pkg/version"
)

const (
	kubeSystemNSName  = "kube-system"
	unknownIdentityID = "00000000-0000-0000-0000-000000000000"

	// Source: https://github.com/kubernetes/kubernetes/blob/v1.25.4/cmd/kubeadm/app/constants/constants.go
	// The labels were copied as it is problematic to add k8s.io/kubernetes dependency: https://github.com/kubernetes/kubernetes/issues/79384
	controlPlaneNodeLabel           = "node-role.kubernetes.io/control-plane"
	deprecatedControlPlaneNodeLabel = "node-role.kubernetes.io/master"

	defaultTimeWindowInHours = 1
)

var (
	// APIKey contains the API key for external analytics service. It is set during application build.
	APIKey string
)

var _ Reporter = &SegmentReporter{}

type BatchedDataStore interface {
	AddSourceEvent(event batched.SourceEvent)
	HeartbeatProperties() batched.HeartbeatProperties
	IncrementTimeWindowInHours()
	Reset()
}

type pluginReport struct {
	Name string
	Type plugin.Type
	RBAC *config.PolicyRule
}

// SegmentReporter is a default Reporter implementation that uses Twilio Segment.
type SegmentReporter struct {
	log logrus.FieldLogger
	cli segment.Client

	identity *Identity

	batchedData  BatchedDataStore
	tickDuration time.Duration
}

// NewSegmentReporter creates a new SegmentReporter instance.
func NewSegmentReporter(log logrus.FieldLogger, cli segment.Client) *SegmentReporter {
	return &SegmentReporter{
		log:          log,
		cli:          cli,
		batchedData:  batched.NewData(defaultTimeWindowInHours),
		tickDuration: defaultTimeWindowInHours * time.Hour,
	}
}

// RegisterCurrentIdentity loads the current anonymous identity and registers it.
func (r *SegmentReporter) RegisterCurrentIdentity(ctx context.Context, k8sCli kubernetes.Interface, remoteDeployID string) error {
	currentIdentity, err := r.load(ctx, k8sCli)
	if err != nil {
		return fmt.Errorf("while loading current identity: %w", err)
	}

	if remoteDeployID != "" {
		currentIdentity.DeploymentID = remoteDeployID
	}

	err = r.registerIdentity(currentIdentity)
	if err != nil {
		return fmt.Errorf("while registering identity: %w", err)
	}

	return nil
}

// ReportCommand reports a new executed command. The command should be anonymized before using this method.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportCommand(in ReportCommandInput) error {
	return r.reportEvent("Command executed", map[string]interface{}{
		"platform": in.Platform,
		"command":  in.Command,
		"plugin":   in.PluginName,
		"origin":   in.Origin,
		"filtered": in.WithFilter,
	})
}

// ReportBotEnabled reports an enabled bot.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportBotEnabled(platform config.CommPlatformIntegration, commGroupIdx int) error {
	return r.reportEvent("Integration enabled", map[string]interface{}{
		"platform":             platform,
		"type":                 config.BotIntegrationType,
		"communicationGroupID": commGroupIdx,
	})
}

// ReportPluginsEnabled reports plugins enabled.
func (r *SegmentReporter) ReportPluginsEnabled(executors map[string]config.Executors, sources map[string]config.Sources) error {
	pluginsConfig := make(map[string]interface{})
	for _, values := range executors {
		r.generatePluginsReport(pluginsConfig, values.Plugins, plugin.TypeExecutor)
	}
	for _, values := range sources {
		r.generatePluginsReport(pluginsConfig, values.Plugins, plugin.TypeSource)
	}
	return r.reportEvent("Plugin enabled", pluginsConfig)
}

// ReportSinkEnabled reports an enabled sink.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportSinkEnabled(platform config.CommPlatformIntegration, commGroupIdx int) error {
	return r.reportEvent("Integration enabled", map[string]interface{}{
		"platform":             platform,
		"type":                 config.SinkIntegrationType,
		"communicationGroupID": commGroupIdx,
	})
}

// ReportHandledEventSuccess reports a successfully handled event using a given communication platform.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportHandledEventSuccess(event ReportEventInput) error {
	r.batchedData.AddSourceEvent(batched.SourceEvent{
		IntegrationType:       event.IntegrationType,
		Platform:              event.Platform,
		PluginName:            event.PluginName,
		AnonymizedEventFields: event.AnonymizedEventFields,
		Success:               true,
	})

	return nil
}

// ReportHandledEventError reports a failure while handling event using a given communication platform.
// The RegisterCurrentIdentity needs to be called first.
func (r *SegmentReporter) ReportHandledEventError(event ReportEventInput, err error) error {
	if err == nil {
		return nil
	}

	r.batchedData.AddSourceEvent(batched.SourceEvent{
		IntegrationType:       event.IntegrationType,
		Platform:              event.Platform,
		PluginName:            event.PluginName,
		AnonymizedEventFields: event.AnonymizedEventFields,
		Success:               false,
		Error:                 ptr.FromType(err.Error()),
	})

	return nil
}

// ReportFatalError reports a fatal app error.
// It doesn't need a registered identity.
func (r *SegmentReporter) ReportFatalError(err error) error {
	if err == nil {
		return nil
	}

	properties := map[string]interface{}{
		"error": err.Error(),
	}

	var anonymousID string
	if r.identity != nil {
		anonymousID = r.identity.AnonymousID
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

// Run runs the reporter.
func (r *SegmentReporter) Run(ctx context.Context) error {
	r.log.Debug("Running heartbeat reporting...")

	ticker := time.NewTicker(r.tickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			err := r.reportHeartbeatEvent()
			if err != nil {
				return fmt.Errorf("while reporting heartbeat event: %w", err)
			}

			return nil
		case <-ticker.C:
			err := r.reportHeartbeatEvent()
			if err != nil {
				r.log.WithError(err).Error("Failed to report heartbeat event")
				r.batchedData.IncrementTimeWindowInHours()
				continue
			}

			r.batchedData.Reset()
		}
	}
}

// Close cleans up the reporter resources.
func (r *SegmentReporter) Close() error {
	r.log.Info("Closing...")
	return r.cli.Close()
}

func (r *SegmentReporter) reportHeartbeatEvent() error {
	r.log.Debug("Reporting heartbeat event...")
	heartbeatProps := r.batchedData.HeartbeatProperties()

	// we can't use mapstructure because of this missing feature: https://github.com/mitchellh/mapstructure/issues/249
	bytes, err := json.Marshal(heartbeatProps)
	if err != nil {
		return fmt.Errorf("while marshalling heartbeat properties: %w", err)
	}

	var props map[string]interface{}
	err = json.Unmarshal(bytes, &props)
	if err != nil {
		return fmt.Errorf("while unmarshalling heartbeat properties: %w", err)
	}

	return r.reportEvent("Heartbeat", props)
}

func (r *SegmentReporter) reportEvent(event string, properties map[string]interface{}) error {
	if r.identity == nil {
		return errors.New("identity needs to be registered first")
	}

	err := r.cli.Enqueue(segment.Track{
		AnonymousId: r.identity.AnonymousID,
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
		AnonymousId: identity.AnonymousID,
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
		AnonymousID:           clusterID,
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
		val, ok := item.Labels[controlPlaneNodeLabel]
		if ok || val == "true" {
			controlPlaneNodesCount++
			continue
		}
		val, ok = item.Labels[deprecatedControlPlaneNodeLabel]
		if ok || val == "true" {
			controlPlaneNodesCount++
			continue
		}

		workerNodesCount++
	}

	return workerNodesCount, controlPlaneNodesCount, nil
}

func (r *SegmentReporter) generatePluginsReport(pluginsConfig map[string]interface{}, plugins config.Plugins, pluginType plugin.Type) {
	for name, pluginValue := range plugins {
		if !pluginValue.Enabled {
			continue
		}
		pluginsConfig[name] = pluginReport{
			Name: name,
			Type: pluginType,
			RBAC: r.getAnonymizedRBAC(pluginValue.Context.RBAC),
		}
	}
}

func (r *SegmentReporter) getAnonymizedRBAC(rbac *config.PolicyRule) *config.PolicyRule {
	rbac.Group.Prefix = r.anonymizedValue(rbac.Group.Prefix)
	for key, name := range rbac.Group.Static.Values {
		rbac.Group.Static.Values[key] = r.anonymizedValue(name)
	}

	rbac.User.Prefix = r.anonymizedValue(rbac.User.Prefix)
	rbac.User.Static.Value = r.anonymizedValue(rbac.User.Static.Value)
	return rbac
}

func (r *SegmentReporter) anonymizedValue(value string) string {
	if value == "" || value == config.RBACDefaultGroup || value == config.RBACDefaultUser {
		return value
	}
	return "***"
}
