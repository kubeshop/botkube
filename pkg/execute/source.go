package execute

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

var (
	sourceFeatureName = FeatureName{
		Name:    "source",
		Aliases: []string{"sources", "src"},
	}
)

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
func (e *SourceExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandList: e.List,
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

type source struct {
	enabled     bool
	displayName string
}

// TabularOutput sorts source groups by key and returns a printable table
func (e *SourceExecutor) TabularOutput(bindings []string) string {
	var keys []string
	sources := make(map[string]source)
	for _, b := range bindings {
		s := e.cfg.Sources[b]
		if len(s.Plugins) > 0 {
			for name, plugin := range s.Plugins {
				keys = append(keys, name)
				sources[name] = source{enabled: plugin.Enabled, displayName: s.DisplayName}
			}
		} else {
			keys = append(keys, b)
			sources[b] = source{enabled: true, displayName: s.DisplayName}
		}
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tENABLED\tDISPLAY NAME")
	sort.Strings(keys)
	for _, k := range keys {
		s := sources[k]
		fmt.Fprintf(w, "%s\t%t\t%s\n", k, s.enabled, s.displayName)
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
