package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	feedbackResourcesNames = noResourceNames
)

// FeedbackExecutor executes all commands that are related to feedback.
type FeedbackExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
}

// NewFeedbackExecutor returns a new FeedbackExecutor instance
func NewFeedbackExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter) *FeedbackExecutor {
	return &FeedbackExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
	}
}

// ResourceNames returns slice of resources the executor supports
func (e *FeedbackExecutor) ResourceNames() []string {
	return feedbackResourcesNames
}

// Commands returns slice of commands the executor supports
func (e *FeedbackExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandFeedback: e.Feedback,
	}
}

// Feedback responds with a feedback form URL
func (e *FeedbackExecutor) Feedback(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	e.reportCommand(cmdCtx.Args[0], cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	return interactive.Feedback(), nil
}

func (e *FeedbackExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting feedback command: %s", err.Error())
	}
}
