package execute

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/dustin/go-humanize/english"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

const (
	editedSourcesMsgFmt              = ":white_check_mark: %s adjusted the Botkube notifications settings to %s messages for this channel. Expect Botkube reload in a few seconds..."
	editedSourcesMsgWithoutReloadFmt = ":white_check_mark: %s adjusted the Botkube notifications settings to %s messages.\nAs the Config Watcher is disabled, you need to restart Botkube manually to apply the changes."
	unknownSourcesMsgFmt             = ":exclamation: The %s %s not found in configuration. To learn how to add custom source, visit https://docs.botkube.io/configuration/source."
)

var (
	sourceBindingFeatureName = FeatureName{
		Name:    "sourcebinding",
		Aliases: []string{"sourcebindings", "sb"},
	}
)

// FeatureName returns the name and aliases of the feature provided by this executor
func (e *SourceBindingExecutor) FeatureName() FeatureName {
	return sourceBindingFeatureName
}

// Commands returns slice of commands the executor supports
func (e *SourceBindingExecutor) Commands() map[command.Verb]CommandFn {
	return map[command.Verb]CommandFn{
		command.EditVerb:   e.Edit,
		command.StatusVerb: e.Status,
	}
}

// BindingsStorage provides functionality to persist source binding for a given channel.
type BindingsStorage interface {
	PersistSourceBindings(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, sourceBindings []string) error
}

// SourceBindingExecutor provides functionality to run all Botkube SourceBinding related commands.
type SourceBindingExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        BindingsStorage
	sources           map[string]string
	cfg               config.Config
}

// NewSourceBindingExecutor returns a new SourceBindingExecutor instance.
func NewSourceBindingExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfgManager BindingsStorage, cfg config.Config) *SourceBindingExecutor {
	normalizedSource := map[string]string{}
	for key, item := range cfg.Sources {
		displayName := item.DisplayName
		if displayName == "" {
			displayName = key // fallback to key
		}
		normalizedSource[key] = displayName
	}

	return &SourceBindingExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfgManager:        cfgManager,
		sources:           normalizedSource,
		cfg:               cfg,
	}
}

// Status returns all sources per given channel
func (e *SourceBindingExecutor) Status(_ context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	sources := e.currentlySelectedOptions(cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation.ID)
	if len(sources) == 0 {
		return interactive.CoreMessage{}, nil
	}
	sort.Strings(sources)

	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "NAME\tDISPLAY NAME")
	for _, name := range sources {
		description := ""
		if s, ok := e.cfg.Sources[name]; ok {
			description = s.DisplayName
		}
		fmt.Fprintf(w, "%s\t%s\n", name, description)
	}
	w.Flush()
	return respond(buf.String(), cmdCtx), nil
}

// Edit executes the edit command based on args.
func (e *SourceBindingExecutor) Edit(ctx context.Context, cmdCtx CommandContext) (interactive.CoreMessage, error) {
	var empty interactive.CoreMessage

	if len(cmdCtx.Args) < 2 {
		return empty, errInvalidCommand
	}

	var (
		cmdName = cmdCtx.Args[0]
		cmdVerb = cmdCtx.Args[1]
		cmdArgs = cmdCtx.Args[2:]
	)

	defer func() {
		cmdToReport := fmt.Sprintf("%s %s", cmdName, cmdVerb)
		err := e.analyticsReporter.ReportCommand(cmdCtx.Platform, cmdToReport, cmdCtx.Conversation.CommandOrigin, false)
		if err != nil {
			e.log.Errorf("while reporting edit command: %s", err.Error())
		}
	}()

	msg, err := e.editSourceBindingHandler(ctx, cmdArgs, cmdCtx.CommGroupName, cmdCtx.Platform, cmdCtx.Conversation, cmdCtx.User, cmdCtx.BotName)
	if err != nil {
		return empty, err
	}
	return msg, nil
}

func (e *SourceBindingExecutor) editSourceBindingHandler(ctx context.Context, cmdArgs []string, commGroupName string, platform config.CommPlatformIntegration, conversation Conversation, userID, botName string) (interactive.CoreMessage, error) {
	var empty interactive.CoreMessage

	sourceBindings, err := e.normalizeSourceItems(cmdArgs)
	if err != nil {
		return empty, fmt.Errorf("while normalizing source args: %w", err)
	}

	if len(sourceBindings) == 0 {
		selectedOptions := e.mapToOptions(e.currentlySelectedOptions(commGroupName, platform, conversation.ID))
		return interactive.CoreMessage{
			Header: "Adjust notifications",
			Message: api.Message{
				Type:              api.PopupMessage,
				OnlyVisibleForYou: true,
				Sections: []api.Section{
					{
						MultiSelect: api.MultiSelect{
							Name: "Adjust notifications",
							Description: api.Body{
								Plaintext: "Select notification sources.",
							},
							Command:        fmt.Sprintf("%s %s", botName, "edit SourceBindings"),
							Options:        e.allOptions(),
							InitialOptions: selectedOptions,
						},
					},
				},
			},
		}, nil
	}

	unknown := e.getUnknownInputSourceBindings(sourceBindings)
	if len(unknown) > 0 {
		return e.generateUnknownMessage(unknown), nil
	}

	err = e.cfgManager.PersistSourceBindings(ctx, commGroupName, platform, conversation.Alias, sourceBindings)
	if err != nil {
		return empty, fmt.Errorf("while persisting source bindings configuration: %w", err)
	}

	names := e.mapToDisplayNames(sourceBindings)
	names = e.quoteEachItem(names)
	sourceList := english.OxfordWordSeries(names, "and")
	if userID == "" {
		userID = "Anonymous"
	}

	return interactive.CoreMessage{
		Description: e.getEditedSourceBindingsMsg(userID, sourceList),
	}, nil
}

func (e *SourceBindingExecutor) getEditedSourceBindingsMsg(userID, sourceList string) string {
	if !e.cfg.ConfigWatcher.Enabled {
		return fmt.Sprintf(editedSourcesMsgWithoutReloadFmt, userID, sourceList)
	}

	return fmt.Sprintf(editedSourcesMsgFmt, userID, sourceList)
}

func (e *SourceBindingExecutor) generateUnknownMessage(unknown []string) interactive.CoreMessage {
	list := english.OxfordWordSeries(e.quoteEachItem(unknown), "and")
	word := english.PluralWord(len(unknown), "source was", "sources were")
	return interactive.CoreMessage{
		Description: fmt.Sprintf(unknownSourcesMsgFmt, list, word),
	}
}

func (e *SourceBindingExecutor) currentlySelectedOptions(commGroupName string, platform config.CommPlatformIntegration, conversationID string) []string {
	switch platform {
	case config.SlackCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Slack.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return channel.Bindings.Sources
		}
	case config.SocketSlackCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].SocketSlack.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return channel.Bindings.Sources
		}
	case config.MattermostCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Mattermost.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return channel.Bindings.Sources
		}
	case config.DiscordCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Mattermost.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return channel.Bindings.Sources
		}
	case config.TeamsCommPlatformIntegration:
		return e.cfg.Communications[commGroupName].Teams.Bindings.Sources
	}
	return nil
}

func (e *SourceBindingExecutor) mapToDisplayNames(in []string) []string {
	var out []string
	for _, key := range in {
		out = append(out, e.sources[key])
	}
	return out
}

func (e *SourceBindingExecutor) mapToOptions(in []string) []api.OptionItem {
	var options []api.OptionItem
	for _, key := range in {
		displayName, found := e.sources[key]
		if !found {
			continue
		}
		options = append(options, api.OptionItem{
			Name:  displayName,
			Value: key,
		})
	}
	return options
}

func (e *SourceBindingExecutor) allOptions() []api.OptionItem {
	var options []api.OptionItem
	for key, displayName := range e.sources {
		options = append(options, api.OptionItem{
			Name:  displayName,
			Value: key,
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Value < options[j].Value
	})

	return options
}

func (*SourceBindingExecutor) normalizeSourceItems(args []string) ([]string, error) {
	var out []string
	for _, item := range args {
		// Case: "foo,baz,bar"
		item, err := removeQuotationMarks(item)
		if err != nil {
			return nil, err
		}

		// Case: foo, baz, bar
		item = strings.ReplaceAll(item, " ", "")

		// Case: `foo`, `baz`, bar
		item = strings.ReplaceAll(item, "`", "")

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

	return out, nil
}

func (e *SourceBindingExecutor) getUnknownInputSourceBindings(sources []string) []string {
	var out []string
	for _, item := range sources {
		_, found := e.sources[item]
		if found {
			continue
		}
		out = append(out, item)
	}
	return out
}

func (*SourceBindingExecutor) quoteEachItem(in []string) []string {
	for idx := range in {
		in[idx] = fmt.Sprintf("`%s`", in[idx])
	}
	return in
}

func isQuotationMark(r rune) bool {
	return unicode.Is(unicode.Quotation_Mark, r)
}

func removeQuotationMarks(in string) (string, error) {
	result, _, err := transform.String(runes.Remove(runes.Predicate(isQuotationMark)), in)
	return result, err
}
