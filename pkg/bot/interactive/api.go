package interactive

import (
	"context"

	"github.com/kubeshop/botkube/pkg/config"
)

// Bot represents bidirectional integration.
type Bot interface {
	BotName() string
	IntegrationName() config.CommPlatformIntegration
}

// Interactive represents interactive bidirectional integration.
type Interactive interface {
	Bot

	// SendInteractiveMessage sends message with interactive sections.
	SendInteractiveMessage(context.Context, Message) error
}
