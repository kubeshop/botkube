package notifier

import (
	"context"
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

// Bot sends event notifications and messages on the communication channels.
type Bot interface {

	// SendMessageToAll is used for notifying about Botkube start/stop listening, possible Botkube upgrades and other events.
	// Some integrations may decide to ignore such messages and have SendMessage method no-op.
	// TODO: Consider option per channel to turn on/off "announcements" (Botkube start/stop/upgrade, notify/config change).
	SendMessageToAll(context.Context, interactive.CoreMessage) error

	// SendMessage sends a generic message for a given source bindings.
	SendMessage(context.Context, interactive.CoreMessage, []string) error

	// IntegrationName returns a name of a given communication platform.
	IntegrationName() config.CommPlatformIntegration

	// Type returns a given integration type. See config.IntegrationType for possible integration types.
	Type() config.IntegrationType
}

// SendPlaintextMessage sends a plaintext message to specified providers.
func SendPlaintextMessage(ctx context.Context, notifiers []Bot, msg string) error {
	if msg == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// Send message over notifiers
	for _, n := range notifiers {
		err := n.SendMessageToAll(ctx, interactive.CoreMessage{
			Message: api.Message{
				BaseBody: api.Body{
					Plaintext: msg,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("error occurred while sending message: \n\t%w", err)
		}
	}

	return nil
}
