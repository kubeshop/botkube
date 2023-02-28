package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	feedbackFeatureName = FeatureName{Name: noFeature}
)

// FeedbackExecutor executes all commands that are related to feedback.
type FeedbackExecutor struct {
	log logrus.FieldLogger
}

// NewFeedbackExecutor returns a new FeedbackExecutor instance
func NewFeedbackExecutor(log logrus.FieldLogger) *FeedbackExecutor {
	return &FeedbackExecutor{
		log: log,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *FeedbackExecutor) FeatureName() FeatureName {
	return feedbackFeatureName
}

// Commands returns slice of commands the executor supports
func (e *FeedbackExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.FeedbackVerb: e.Feedback,
	}
}

// Feedback responds with a feedback form URL
func (e *FeedbackExecutor) Feedback(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	return interactive.Feedback(), nil
}
