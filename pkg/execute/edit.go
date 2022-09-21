package execute

import (
	"fmt"
	"strings"

	"github.com/dustin/go-humanize/english"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	editedSourcesMsgFmt = ":white_check_mark: %s adjusted the BotKube notifications settings to %s messages."
)

// EditResource defines the name of editable resource
type EditResource string

const (
	// SourceBindings define name of source binding resource
	SourceBindings EditResource = "SourceBindings"
)

// Key returns normalized edit resource name.
func (e EditResource) Key() string {
	return strings.ToLower(string(e))
}

// BindingsStorage provides functionality to persist source binding for a given channel.
type BindingsStorage interface {
	PersistSourceBindings(commGroupName string, platform config.CommPlatformIntegration, channelName string, sourceBindings []string) error
}

// EditExecutor provides functionality to run all BotKube edit related commands.
type EditExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        BindingsStorage
}

// NewEditExecutor returns a new EditExecutor instance.
func NewEditExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfgManager BindingsStorage) *EditExecutor {
	return &EditExecutor{log: log, analyticsReporter: analyticsReporter, cfgManager: cfgManager}
}

// Do executes a given edit command based on args.
func (e *EditExecutor) Do(args []string, commGroupName string, platform config.CommPlatformIntegration, conversationID, userID string) (interactive.Message, error) {
	var empty interactive.Message

	if len(args) < 2 {
		return empty, errInvalidCommand
	}

	var (
		cmdName = args[0]
		cmdVerb = args[1]
		cmdArgs = args[2:]
	)

	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", cmdName, cmdVerb)
		err := e.analyticsReporter.ReportCommand(platform, cmdToReport)
		if err != nil {
			e.log.Errorf("while reporting edit command: %s", err.Error())
		}
	}()

	cmds := executorsRunner{
		SourceBindings.Key(): func() (interactive.Message, error) {
			sourceBindings := e.normalizeSourceItems(cmdArgs)
			if len(sourceBindings) == 0 {
				return empty, errInvalidCommand
			}

			err := e.cfgManager.PersistSourceBindings(commGroupName, platform, conversationID, sourceBindings)
			if err != nil {
				return empty, fmt.Errorf("while persisting source bindings configuration: %w", err)
			}

			sourceList := english.OxfordWordSeries(sourceBindings, "and")
			return interactive.Message{
				Base: interactive.Base{
					Description: fmt.Sprintf(editedSourcesMsgFmt, userID, sourceList),
				},
			}, nil
		},
	}

	msg, err := cmds.SelectAndRun(cmdVerb)
	if err != nil {
		cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
		return empty, err
	}
	return msg, nil
}

func (*EditExecutor) normalizeSourceItems(args []string) []string {
	var out []string
	for _, item := range args {
		// Case: "foo,baz,bar"
		item = strings.Trim(item, `"`)

		// Case: foo, baz, bar
		item = strings.ReplaceAll(item, " ", "")

		// Case: foo, baz
		//       bar
		item = strings.ReplaceAll(item, "\n", "")

		// Case: foo,baz,bar
		candidates := strings.Split(item, ",")

		// Filter out all empty items.
		// Case: foo,,baz,
		for _, i := range candidates {
			if i == "" {
				continue
			}
			out = append(out, i)
		}
	}

	return out
}
