package teamsxold

import (
	"errors"

	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// MessageOptionMutator modifies MessageOptions.
type MessageOptionMutator func(*MessageOptions)

// MessageOptions contains necessary configuration for sending a message.
type MessageOptions struct {
	Message interactive.CoreMessage
	ConvRef ConversationReference
}

// Validate checks if MessageOptions is valid.
func (m *MessageOptions) Validate() error {
	issues := multierror.New()
	if m.Message.IsEmpty() {
		issues = multierror.Append(issues, errors.New("message cannot be empty"))
	}
	if m.ConvRef.Conversation.ID == "" {
		issues = multierror.Append(issues, errors.New("conversation id cannot be empty"))
	}

	return issues.ErrorOrNil()
}

// WithCoreMessage sets the CoreMessage in MessageOptions.
func WithCoreMessage(in interactive.CoreMessage) func(opts *MessageOptions) {
	return func(opts *MessageOptions) {
		opts.Message = in
	}
}

// WithMessage sets the message in MessageOptions.
func WithMessage(in api.Message) func(opts *MessageOptions) {
	return WithCoreMessage(interactive.CoreMessage{Message: in})
}

// WithConvRefFromActivity sets ConversationReference from Activity.
func WithConvRefFromActivity(act schema.Activity) func(opts *MessageOptions) {
	return func(opts *MessageOptions) {
		user := act.From
		if act.Type == schema.Invoke {
			user = act.Recipient
		}

		opts.ConvRef = ConversationReference{
			ActivityID:   act.ID,
			User:         user,
			Conversation: act.Conversation,
			ServiceURL:   act.ServiceURL,
			ReplyToID:    act.ReplyToID,
		}
	}
}

// ConversationReference contains reference information for a conversation.
type ConversationReference struct {
	ActivityID   string
	User         schema.ChannelAccount
	Conversation schema.ConversationAccount
	ServiceURL   string
	ReplyToID    string
}

// WithServiceURLAndConvID sets ServiceURL and Conversation ID in MessageOptions.
func WithServiceURLAndConvID(url, convID string) func(opts *MessageOptions) {
	return func(opts *MessageOptions) {
		opts.ConvRef = ConversationReference{
			Conversation: schema.ConversationAccount{
				ID: convID,
			},
			ServiceURL: url,
		}
	}
}
