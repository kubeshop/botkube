package controller

import (
	"context"
	"fmt"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// Notifier sends event notifications and messages on the communication channels.
type Notifier interface {
	SendEvent(context.Context, events.Event) error
	SendMessage(context.Context, string) error
	IntegrationName() config.CommPlatformIntegration
	Type() config.IntegrationType
}

func sendMessageToNotifiers(ctx context.Context, notifiers []Notifier, msg string) error {
	if msg == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// Send message over notifiers
	for _, n := range notifiers {
		err := n.SendMessage(ctx, msg)
		if err != nil {
			return fmt.Errorf("while sending message: %w", err)
		}
	}

	return nil
}
