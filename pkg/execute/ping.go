package execute

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	pingFeatureName = FeatureName{Name: noFeature}
)

// PingExecutor executes all commands that are related to ping.
type PingExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	botkubeVersion    string
}

// NewPingExecutor returns a new PingExecutor instance.
func NewPingExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, botkubeVersion string) *PingExecutor {
	return &PingExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		botkubeVersion:    botkubeVersion,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *PingExecutor) FeatureName() FeatureName {
	return pingFeatureName
}

// Commands returns slice of commands the executor supports
func (e *PingExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.PingVerb: e.Ping,
	}
}

// Ping responds with "pong" to the ping command
func (e *PingExecutor) Ping(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	cmdVerb, _ := parseCmdVerb(cmdCtx.Args)
	e.log.Debugf("Sending pong to %s", cmdCtx.Conversation.ID)
	e.reportCommand(cmdVerb, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	return respond(fmt.Sprintf("pong\n\n%s", e.botkubeVersion), cmdCtx), nil
}

func (e *PingExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting ping command: %s", err.Error())
	}
}
