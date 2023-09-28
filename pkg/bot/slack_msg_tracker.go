package bot

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const (
	msgReceivedEmoji  = "eyes"
	msgProcessedEmoji = "white_check_mark"
)

// SlackReactionClient defines the interface for managing reactions on Slack messages.
type SlackReactionClient interface {
	AddReaction(name string, item slack.ItemRef) error
	RemoveReaction(name string, item slack.ItemRef) error
}

// SlackMessageStatusTracker marks messages with emoji for easy tracking the status of Slack messages.
type SlackMessageStatusTracker struct {
	log    logrus.FieldLogger
	client SlackReactionClient
}

// NewSlackMessageStatusTracker creates a new instance of SlackMessageStatusTracker.
func NewSlackMessageStatusTracker(log logrus.FieldLogger, client SlackReactionClient) *SlackMessageStatusTracker {
	return &SlackMessageStatusTracker{log: log, client: client}
}

// GetMsgRef retrieves the Slack item reference for a given event.
// It returns nil if the message doesn't support reactions or lacks necessary information.
func (b *SlackMessageStatusTracker) GetMsgRef(event slackMessage) *slack.ItemRef {
	// We may not have it when it is visible only to a user,
	// or it was a modal that has only a trigger ID.
	if event.EventTimeStamp == "" || event.Channel == "" {
		b.log.WithField("commandOrigin", event.CommandOrigin).Debug("Message doesn't support reactions. Skipping...")
		return nil
	}
	ref := slack.NewRefToMessage(event.Channel, event.EventTimeStamp)
	return &ref
}

// MarkAsReceived marks a message as received by adding the "eyes" reaction.
// If msgRef is nil, no action is performed.
func (b *SlackMessageStatusTracker) MarkAsReceived(msgRef *slack.ItemRef) {
	if msgRef == nil {
		return
	}
	err := b.client.AddReaction(msgReceivedEmoji, *msgRef)
	b.handleReactionError(err, "received")
}

// MarkAsProcessed marks a message as processed by removing the "eyes" reaction and adding the "heavy_check_mark" reaction.
// If msgRef is nil, no action is performed.
func (b *SlackMessageStatusTracker) MarkAsProcessed(msgRef *slack.ItemRef) {
	b.MarkAsProcessedWithCustomEmoji(msgRef, msgProcessedEmoji)
}

func (b *SlackMessageStatusTracker) MarkAsProcessedWithCustomEmoji(msgRef *slack.ItemRef, emoji string) {
	if msgRef == nil {
		return
	}

	_ = b.client.RemoveReaction(msgReceivedEmoji, *msgRef) // The reaction may be missing as there was an error earlier.

	if emoji == "" {
		return
	}

	err := b.client.AddReaction(emoji, *msgRef)
	b.handleReactionError(err, "processed")
}

func (b *SlackMessageStatusTracker) handleReactionError(err error, ctx string) {
	logMsg := fmt.Sprintf("Cannot mark message as %s.", ctx)
	switch terr := err.(type) {
	case nil:
		// No error occurred, do nothing.
	case slack.SlackErrorResponse:
		b.log.WithFields(logrus.Fields{
			"messages": terr.ResponseMetadata.Messages,
		}).WithError(err).Warn(logMsg)
	default:
		b.log.WithError(err).Warn(logMsg)
	}
}
