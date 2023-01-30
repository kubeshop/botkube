package execute

import (
	"bytes"
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/alias"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/maputil"
)

var _ CommandExecutor = &AliasExecutor{}

const aliasesForCurrentBindingsMsg = "Only showing aliases for executors enabled for this channel."

var featureName = FeatureName{
	Name:    "alias",
	Aliases: []string{"aliases", "als"},
}

type AliasExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfg               config.Config
}

// NewAliasExecutor
func NewAliasExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfg config.Config) *AliasExecutor {
	return &AliasExecutor{log: log, analyticsReporter: analyticsReporter, cfg: cfg}
}

// Commands returns slice of commands the executor supports.
func (e *AliasExecutor) Commands() map[CommandVerb]CommandFn {
	return map[CommandVerb]CommandFn{
		CommandList: e.List,
	}
}

// List returns a tabular representation of aliases.
func (e *AliasExecutor) List(_ context.Context, cmdCtx CommandContext) (interactive.Message, error) {
	cmdVerb, cmdRes := parseCmdVerb(cmdCtx.Args)
	defer e.reportCommand(cmdVerb, cmdRes, cmdCtx.Conversation.CommandOrigin, cmdCtx.Platform)
	e.log.Debug("Listing aliases...")
	outMsg := respond(e.getTabularOutput(cmdCtx.Conversation.ExecutorBindings), cmdCtx)
	outMsg.Sections = []interactive.Section{
		{
			Context: []interactive.ContextItem{
				{Text: aliasesForCurrentBindingsMsg},
			},
		},
	}

	return outMsg, nil
}

// FeatureName returns the name and aliases of the feature provided by this executor.
func (e *AliasExecutor) FeatureName() FeatureName {
	return featureName
}

func (e *AliasExecutor) reportCommand(cmdVerb, cmdRes string, commandOrigin command.Origin, platform config.CommPlatformIntegration) {
	cmdToReport := fmt.Sprintf("%s %s", cmdVerb, cmdRes)
	err := e.analyticsReporter.ReportCommand(platform, cmdToReport, commandOrigin, false)
	if err != nil {
		e.log.Errorf("while reporting executor command: %s", err.Error())
	}
}

func (e *AliasExecutor) getTabularOutput(bindings []string) string {
	aliasesToDisplay := make(map[string]config.Alias)

	aliasesCfg := e.cfg.Aliases
	executors := executorsForBindings(e.cfg.Executors, bindings)
	for exName, enabled := range executors {
		if !enabled {
			continue
		}

		aliasesForPrefix := alias.ListForExecutorPrefix(exName, aliasesCfg)
		for _, aliasName := range aliasesForPrefix {
			aliasesToDisplay[aliasName] = aliasesCfg[aliasName]
		}
	}

	if len(aliasesToDisplay) == 0 {
		return "No aliases found for current conversation."
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ALIAS\tCOMMAND\tDISPLAY NAME")
	for _, aliasName := range maputil.SortKeys(aliasesToDisplay) {
		aliasCfg := aliasesCfg[aliasName]
		fmt.Fprintf(w, "%s\t%s\t%s\n", aliasName, aliasCfg.Command, aliasCfg.DisplayName)
	}

	if len(aliasesToDisplay) == 0 {
		fmt.Fprintln(w, "No aliases found for current conversation.")
	}

	w.Flush()
	return buf.String()
}
