package execute

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/maputil"
)

var (
	sourceFeatureName = FeatureName{
		Name:    "source",
		Aliases: []string{"sources", "src"},
	}
)

const kubernetesBuiltinSourceName = "kubernetes"

// SourceExecutor executes all commands that are related to sources.
type SourceExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfg               config.Config
}

// NewSourceExecutor returns a new SourceExecutor instance.
func NewSourceExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfg config.Config) *SourceExecutor {
	return &SourceExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfg:               cfg,
	}
}

// Commands returns slice of commands the executor supports
func (e *SourceExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.ListVerb: e.List,
	}
}

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *SourceExecutor) FeatureName() FeatureName {
	return sourceFeatureName
}

// List returns a tabular representation of Executors
func (e *SourceExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	e.log.Debug("List sources")
	return respond(e.TabularOutput(cmdCtx.Conversation.SourceBindings), cmdCtx), nil
}

// TabularOutput sorts source groups by key and returns a printable table
func (e *SourceExecutor) TabularOutput(bindings []string) string {
	sources := make(map[string]bool)
	for _, b := range bindings {
		s, ok := e.cfg.Sources[b]
		if !ok {
			continue
		}

		for name, plugin := range s.Plugins {
			sources[name] = plugin.Enabled
		}

		// TODO: Remove once we extract the source to a separate plugin
		if !reflect.DeepEqual(s.Kubernetes, config.KubernetesSource{}) {
			sources[kubernetesBuiltinSourceName] = true
		}
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tENABLED")
	for _, key := range maputil.SortKeys(sources) {
		enabled := sources[key]
		fmt.Fprintf(w, "%s\t%t\n", key, enabled)
	}
	w.Flush()
	return buf.String()
}

func (e *SourceExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting source command: %s", err.Error())
	}
}
