package execute

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	versionFeatureName = FeatureName{Name: noFeature}
)

// VersionExecutor executes all commands that are related to version.
type VersionExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	botkubeVersion    string
}

// NewVersionExecutor returns a new VersionExecutor instance
func NewVersionExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, botkubeVersion string) *VersionExecutor {
	return &VersionExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		botkubeVersion:    botkubeVersion,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *VersionExecutor) FeatureName() FeatureName {
	return versionFeatureName
}

// Commands returns slice of commands the executor supports
func (e *VersionExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.VersionVerb: e.Version,
	}
}

// Version responds with k8s and botkube version string
func (e *VersionExecutor) Version(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, _ := parseCmdVerb(cmdCtx.Args)
	e.reportCommand(cmdVerb, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	return respond(e.botkubeVersion, cmdCtx), nil
}

func (e *VersionExecutor) reportCommand(cmdToReport string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting version command: %s", err.Error())
	}
}
