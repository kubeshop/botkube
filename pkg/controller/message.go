package controller

import (
	"context"
	"fmt"

	"github.com/infracloudio/botkube/pkg/notify"
)

func sendMessageToNotifiers(ctx context.Context, notifiers []notify.Notifier, msg string) error {
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
