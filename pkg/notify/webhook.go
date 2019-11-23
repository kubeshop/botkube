package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
)

// Webhook contains URL and ClusterName
type Webhook struct {
	URL         string
	ClusterName string
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
	Level    events.Level     `json:"level"`
	Reason   string           `json:"reason,omitempty"`
	Error    string           `json:"error,omitempty"`
	Messages []string         `json:"messages,omitempty"`
}

// NewWebhook returns new Webhook object
func NewWebhook(c *config.Config) Notifier {
	return &Webhook{
		URL:         c.Communications.Webhook.URL,
		ClusterName: c.Settings.ClusterName,
	}
}

// SendEvent sends event notification to Webhook url
func (w *Webhook) SendEvent(event events.Event) (err error) {

	// set missing cluster name to event object
	event.Cluster = w.ClusterName

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
		EventSummary:    formatShortMessage(event),
		TimeStamp:       event.TimeStamp,
		Recommendations: event.Recommendations,
		Warnings:        event.Warnings,
	}

	err = w.PostWebhook(jsonPayload)
	if err != nil {
		log.Logger.Error(err.Error())
		log.Logger.Debugf("Event Not Sent to Webhook %v", event)
	}

	log.Logger.Debugf("Event successfully sent to Webhook %v", event)
	return nil
}

// SendMessage sends message to Webhook url
func (w *Webhook) SendMessage(msg string) error {
	return nil
}

// PostWebhook posts webhook to listener
func (w *Webhook) PostWebhook(jsonPayload *WebhookPayload) error {

	message, err := json.Marshal(jsonPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", w.URL, bytes.NewBuffer(message))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Error Posting Webhook: %s", string(resp.StatusCode))
	}

	return nil
}
