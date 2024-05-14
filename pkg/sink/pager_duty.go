package sink

import (
	"context"
	"fmt"
	"github.com/kubeshop/botkube/pkg/api"
	"strings"
	"time"

	"github.com/prometheus/common/model"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/kubeshop/botkube/internal/health"
	k8sconfig "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/sliceutil"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

// PagerDuty provides functionality to notify PagerDuty service about new events.
type PagerDuty struct {
	log      logrus.FieldLogger
	reporter AnalyticsReporter

	bindings config.SinkBindings

	status         health.PlatformStatusMsg
	failureReason  health.FailureReasonMsg
	integrationKey string
	clusterName    string
	pagerDutyCli   *pagerduty.Client
}

// PagerDutyPayload contains JSON payload to be sent to PagerDuty API.
type PagerDutyPayload struct {
	Source    string    `json:"source,omitempty"`
	Data      any       `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EventLink represents a single link in a ChangeEvent and Alert event
// https://developer.pagerduty.com/docs/events-api-v2/send-change-events/#the-links-property
type EventLink struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

// NewPagerDuty creates a new PagerDuty instance.
func NewPagerDuty(log logrus.FieldLogger, commGroupIdx int, c config.PagerDuty, clusterName string, reporter AnalyticsReporter) (*PagerDuty, error) {
	notifier := &PagerDuty{
		log:      log,
		reporter: reporter,

		bindings:       c.Bindings,
		clusterName:    clusterName,
		integrationKey: c.IntegrationKey,

		status:        health.StatusUnknown,
		failureReason: "",

		pagerDutyCli: pagerduty.NewClient(""),
	}

	err := reporter.ReportSinkEnabled(notifier.IntegrationName(), commGroupIdx)
	if err != nil {
		log.WithError(err).Error("Failed to report analytics")
	}

	return notifier, nil
}

// SendEvent sends an event to a configured server.
func (w *PagerDuty) SendEvent(ctx context.Context, rawData any, sources []string) error {
	if !w.shouldNotify(sources) {
		return nil
	}

	jsonPayload := &PagerDutyPayload{
		Source:    strings.Join(sources, ","),
		Data:      rawData,
		Timestamp: time.Now(),
	}

	resp, err := w.postAlertEvent(ctx, jsonPayload)
	if err != nil {
		w.setFailureReason(health.FailureReasonConnectionError)
		return fmt.Errorf("while sending message to PagerDuty: %w", err)
	}

	w.markHealthy()
	w.log.WithFields(logrus.Fields{
		"payload":  jsonPayload,
		"response": resp,
	}).Debug("Message successfully sent")

	return nil
}

// IntegrationName describes the notifier integration name.
func (w *PagerDuty) IntegrationName() config.CommPlatformIntegration {
	return config.PagerDutyCommPlatformIntegration
}

// Type describes the notifier type.
func (w *PagerDuty) Type() config.IntegrationType {
	return config.SinkIntegrationType
}

// GetStatus gets sink status.
func (w *PagerDuty) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   w.status,
		Restarts: "0/0",
		Reason:   w.failureReason,
	}
}

func (w *PagerDuty) shouldNotify(sourceBindings []string) bool {
	return sliceutil.Intersect(sourceBindings, w.bindings.Sources)
}

type k8sEventData struct {
	// Fields set by kubernetes source
	Level     k8sconfig.Level
	Type      string
	Kind      string
	Name      string
	Namespace string
	Messages  []string
}

type argoData struct {
	Message api.Message           
}
type eventData struct {
	k8sEventData `mapstructure:",squash"`

	// Fields set by Prometheus
	Annotations model.LabelSet

	// Fields set by ArgoCD
	argoData `mapstructure:",squash"`
}

type eventMeta struct {
	Summary   string
	Component string
	IsAlert   bool
}

func (w *PagerDuty) getEventMeta(in *PagerDutyPayload) eventMeta {
	out := eventMeta{
		Summary: fmt.Sprintf("Event from %s", in.Source),
		IsAlert: true,
	}

	var ev eventData
	err := mapstructure.Decode(in.Data, &ev)
	if err != nil {
		// we failed, so let's treat it as an error
		w.log.WithError(err).Error("Failed to decode event. Forwarding it to PagerDuty as alert.")
		return out
	}

	
	// handle argo
	ev.argoData.Message.Sections[0].Header  // strip emoji
	
	
	
	switch ev.Level {
	case k8sconfig.Info, k8sconfig.Success:
		out.IsAlert = false
	case k8sconfig.Error:
		out.IsAlert = true
	}

	if ev.Kind != "" && ev.Name != "" && ev.Namespace != "" {
		// Logical grouping of components of a service.
		// source: https://developer.pagerduty.com/api-reference/368ae3d938c9e-send-an-event-to-pager-duty
		out.Component = fmt.Sprintf("%s/%s/%s", ev.Kind, ev.Namespace, ev.Name)
	}

	if ev.Messages != nil {
		out.Summary = strings.Join(ev.Messages, "\n")
	}

	if ev.Type != "" {
		out.Summary = fmt.Sprintf("[%s] %s", ev.Type, out.Summary)
	}

	return out
}

func (w *PagerDuty) postAlertEvent(ctx context.Context, in *PagerDutyPayload) (any, error) {
	meta := w.getEventMeta(in)
	if meta.IsAlert {
		return w.triggerAlert(ctx, in, meta)
	}

	return w.triggerChange(ctx, in, meta)
}

func (w *PagerDuty) triggerAlert(ctx context.Context, in *PagerDutyPayload, meta eventMeta) (*pagerduty.V2EventResponse, error) {
	return pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		// required
		RoutingKey: w.integrationKey,
		Action:     "trigger",

		Client:    "Botkube",
		ClientURL: "https://app.botkube.io",

		Payload: &pagerduty.V2Payload{
			// required

			// A brief text summary of the event, used to generate the summaries/titles of any associated alerts.
			// The maximum permitted length of this property is 1024 characters.
			// TODO: improve the summary. Consider using AI to generate it.
			Summary: meta.Summary,
			// The unique location of the affected system, preferably a hostname or FQDN.
			Source: fmt.Sprintf("%s/%s", w.clusterName, in.Source),
			// The perceived severity of the status the event is describing with respect to the affected system. This can be critical, error, warning or info.
			Severity: "error",

			// optional
			Timestamp: in.Timestamp.Format(time.RFC3339),
			// Component of the source machine that is responsible for the event.
			Group: w.clusterName,
			// Component of the source machine that is responsible for the event.
			Component: meta.Component,
			Details:   in,
		},
	})
}

func (w *PagerDuty) triggerChange(ctx context.Context, in *PagerDutyPayload, meta eventMeta) (*pagerduty.ChangeEventResponse, error) {
	customDetails := map[string]any{
		// Component of the source machine that is responsible for the event.
		"group":   w.clusterName,
		"details": in,
	}

	if meta.Component != "" {
		// Component of the source machine that is responsible for the event.
		customDetails["component"] = meta.Component
	}

	return w.pagerDutyCli.CreateChangeEventWithContext(ctx, pagerduty.ChangeEvent{
		// required
		RoutingKey: w.integrationKey,
		Payload: pagerduty.ChangeEventPayload{
			// required

			// A brief text summary of the event, used to generate the summaries/titles of any associated alerts.
			// The maximum permitted length of this property is 1024 characters.
			// TODO: improve the summary. Consider using AI to generate it.
			Summary: meta.Summary,
			// The unique location of the affected system, preferably a hostname or FQDN.
			Source: fmt.Sprintf("%s/%s", w.clusterName, in.Source),

			// optional
			Timestamp:     in.Timestamp.Format(time.RFC3339),
			CustomDetails: customDetails,
		},
	})
}

func (w *PagerDuty) setFailureReason(reason health.FailureReasonMsg) {
	if reason == "" {
		return
	}
	w.status = health.StatusHealthy
	w.failureReason = reason
}

func (w *PagerDuty) markHealthy() {
	w.status = health.StatusHealthy
	w.failureReason = ""
}
