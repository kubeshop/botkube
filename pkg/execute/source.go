package execute

import (
	"bytes"
	"context"
	"fmt"
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

// SourceExecutor executes all commands that are related to sources.
type SourceExecutor struct {
	log logrus.FieldLogger
	cfg config.Config
}

// NewSourceExecutor returns a new SourceExecutor instance.
func NewSourceExecutor(log logrus.FieldLogger, cfg config.Config) *SourceExecutor {
	return &SourceExecutor{
		log: log,
		cfg: cfg,
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
func (e *SourceExecutor) List(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
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
	}

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintf(w, "SOURCE\tENABLED")
	for _, key := range maputil.SortKeys(sources) {
		enabled := sources[key]
		fmt.Fprintf(w, "\n%s\t%t", key, enabled)
	}
	w.Flush()
	return buf.String()
}
