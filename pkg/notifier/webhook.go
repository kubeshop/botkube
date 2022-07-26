package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const defaultHTTPCliTimeout = 30 * time.Second

// Webhook contains URL
type Webhook struct {
	log      logrus.FieldLogger
	reporter SinkAnalyticsReporter

	URL string
}

// WebhookPayload contains json payload to be sent to webhook url
type WebhookPayload struct {
	EventMeta       EventMeta   `json:"meta"`
	EventStatus     EventStatus `json:"status"`
	EventSummary    string      `json:"summary"`
	TimeStamp       time.Time   `json:"timestamp"`
	Recommendations []string    `json:"recommendations,omitempty"`
	Warnings        []string    `json:"warnings,omitempty"`
}

// EventMeta contains the meta data about the event occurred
type EventMeta struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Cluster   string `json:"cluster,omitempty"`
}

// EventStatus contains the status about the event occurred
type EventStatus struct {
	Type     config.EventType `json:"type"`
	Level    config.Level     `json:"level"`
	Reason   string           `json:"reason,omitempty"`
	Error    string           `json:"error,omitempty"`
	Messages []string         `json:"messages,omitempty"`
}

// NewWebhook returns new Webhook object
func NewWebhook(log logrus.FieldLogger, c config.Communications, reporter SinkAnalyticsReporter) (*Webhook, error) {
	whNotifier := &Webhook{
		log:      log,
		reporter: reporter,
		URL:      c.Webhook.URL,
	}

	err := reporter.ReportSinkEnabled(whNotifier.IntegrationName())
	if err != nil {
		return nil, fmt.Errorf("while reporting analytics: %w", err)
	}

	return whNotifier, nil
}

// SendEvent sends event notification to Webhook url
func (w *Webhook) SendEvent(ctx context.Context, event events.Event) (err error) {
	jsonPayload := &WebhookPayload{
		EventMeta: EventMeta{
			Kind:      event.Kind,
			Name:      event.Name,
			Namespace: event.Namespace,
			Cluster:   event.Cluster,
		},
		EventStatus: EventStatus{
			Type:     event.Type,
			Level:    event.Level,
			Reason:   event.Reason,
			Error:    event.Error,
			Messages: event.Messages,
		},
		EventSummary:    FormatShortMessage(event),
		TimeStamp:       event.TimeStamp,
		Recommendations: event.Recommendations,
		Warnings:        event.Warnings,
	}

	err = w.PostWebhook(ctx, jsonPayload)
	if err != nil {
		return fmt.Errorf("while sending event to webhook: %w", err)
	}

	w.log.Debugf("Event successfully sent to Webhook: %+v", event)
	return nil
}

// SendMessage is no-op
func (w *Webhook) SendMessage(_ context.Context, _ string) error {
	return nil
}

// PostWebhook posts webhook to listener
func (w *Webhook) PostWebhook(ctx context.Context, jsonPayload *WebhookPayload) (err error) {
	message, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: defaultHTTPCliTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		deferredErr := resp.Body.Close()
		if deferredErr != nil {
			err = multierror.Append(err, deferredErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error Posting Webhook: %s", fmt.Sprint(resp.StatusCode))
	}

	return nil
}

// IntegrationName describes the notifier integration name.
func (w *Webhook) IntegrationName() config.CommPlatformIntegration {
	return config.WebhookCommPlatformIntegration
}

// Type describes the notifier type.
func (w *Webhook) Type() config.IntegrationType {
	return config.SinkIntegrationType
}
