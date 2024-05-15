package sink

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/health"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

// PagerDuty provides functionality to notify PagerDuty service about new events.
type PagerDuty struct {
	log      logrus.FieldLogger
	reporter AnalyticsReporter

	bindings config.SinkBindings

	integrationKey string
	clusterName    string
	pagerDutyCli   *pagerduty.Client

	status        health.PlatformStatusMsg
	failureReason health.FailureReasonMsg
	statusMux     sync.Mutex
}

// PagerDutyPayload contains JSON payload to be sent to PagerDuty API.
type PagerDutyPayload struct {
	Source    string    `json:"source,omitempty"`
	Data      any       `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EventLink represents a link in a ChangeEvent and Alert event
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

func (w *PagerDuty) getEventMeta(in *PagerDutyPayload) eventMetadata {
	out := eventMetadata{
		Summary: fmt.Sprintf("Event from %s source", in.Source),
		IsAlert: true,
	}

	var ev eventPayload
	err := mapstructure.Decode(in.Data, &ev)
	if err != nil {
		// we failed, so let's treat it as an error
		w.log.WithError(err).Error("Failed to decode event. Forwarding it to PagerDuty as an alert.")
		return out
	}

	if ev.k8sEventPayload.Level != "" {
		return enrichWithK8sEventMetadata(out, ev.k8sEventPayload)
	}

	if !ev.argoPayload.Message.IsEmpty() {
		return enrichWithArgoCDEventMetadata(out, ev.argoPayload)
	}

	if len(ev.prometheusEventPayload.Annotations) > 0 {
		return enrichWithPrometheusEventMetadata(out, ev.prometheusEventPayload)
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

func (w *PagerDuty) triggerAlert(ctx context.Context, in *PagerDutyPayload, meta eventMetadata) (*pagerduty.V2EventResponse, error) {
	return pagerduty.ManageEventWithContext(ctx, pagerduty.V2Event{
		// required
		RoutingKey: w.integrationKey,
		Action:     "trigger",

		Client:    "Botkube",
		ClientURL: "https://app.botkube.io",

		Payload: &pagerduty.V2Payload{
			// required
			Summary: meta.Summary,
			// The unique location of the affected system, preferably a hostname or FQDN.
			Source: fmt.Sprintf("%s/%s", w.clusterName, in.Source),
			// The perceived severity of the status the event is describing with respect to the affected system. This can be critical, error, warning or info.
			Severity: "error",

			// optional
			Timestamp: in.Timestamp.Format(time.RFC3339),
			// Logical grouping of components of a service.
			Group:     w.clusterName,
			Component: meta.Component,
			Details:   in,
		},
	})
}

func (w *PagerDuty) triggerChange(ctx context.Context, in *PagerDutyPayload, meta eventMetadata) (*pagerduty.ChangeEventResponse, error) {
	customDetails := map[string]any{
		"group":   w.clusterName,
		"details": in,
	}

	if meta.Component != "" {
		customDetails["component"] = meta.Component
	}

	return w.pagerDutyCli.CreateChangeEventWithContext(ctx, pagerduty.ChangeEvent{
		// required
		RoutingKey: w.integrationKey,
		Payload: pagerduty.ChangeEventPayload{
			// required
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

	w.statusMux.Lock()
	defer w.statusMux.Unlock()

	w.status = health.StatusUnHealthy
	w.failureReason = reason
}

func (w *PagerDuty) markHealthy() {
	if w.status == health.StatusHealthy {
		return
	}

	w.statusMux.Lock()
	defer w.statusMux.Unlock()

	w.status = health.StatusHealthy
	w.failureReason = ""
}
