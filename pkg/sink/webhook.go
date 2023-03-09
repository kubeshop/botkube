package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const defaultHTTPCliTimeout = 30 * time.Second

// Webhook provides functionality to notify external service about new events.
type Webhook struct {
	log      logrus.FieldLogger
	reporter AnalyticsReporter

	URL      string
	Bindings config.SinkBindings
}

// WebhookPayload contains json payload to be sent to webhook url
type WebhookPayload struct {
	Source    string    `json:"source,omitempty"`
	Data      any       `json:"data,omitempty"`
	TimeStamp time.Time `json:"timeStamp"`
}

// NewWebhook creates a new Webhook instance.
func NewWebhook(log logrus.FieldLogger, c config.Webhook, reporter AnalyticsReporter) (*Webhook, error) {
	whNotifier := &Webhook{
		log:      log,
		reporter: reporter,
		URL:      c.URL,
		Bindings: c.Bindings,
	}

	err := reporter.ReportSinkEnabled(whNotifier.IntegrationName())
	if err != nil {
		return nil, fmt.Errorf("while reporting analytics: %w", err)
	}

	return whNotifier, nil
}

// SendMessageToAll is no-op.
func (w *Webhook) SendMessageToAll(_ context.Context, _ interactive.CoreMessage) error {
	return nil
}

// SendMessage is no-op.
func (w *Webhook) SendMessage(ctx context.Context, _ interactive.CoreMessage, sources []string, rawData any) error {
	jsonPayload := &WebhookPayload{
		Source: strings.Join(sources, ","),
		Data:   rawData,
	}

	err := w.PostWebhook(ctx, jsonPayload)
	if err != nil {
		return fmt.Errorf("while sending message to webhook: %w", err)
	}

	w.log.Debugf("Message successfully sent to Webhook: %+v", rawData)
	return nil
}

// PostWebhook posts webhook to listener
func (w *Webhook) PostWebhook(ctx context.Context, jsonPayload *WebhookPayload) (err error) {
	message, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewBuffer(message))
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
