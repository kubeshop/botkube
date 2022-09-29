package execute

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/dustin/go-humanize/english"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

const (
	editedSourcesMsgFmt              = ":white_check_mark: %s adjusted the BotKube notifications settings to %s messages. Expect BotKube reload in a few seconds..."
	editedSourcesMsgWithoutReloadFmt = ":white_check_mark: %s adjusted the BotKube notifications settings to %s messages.\nAs the Config Watcher is disabled, you need to restart BotKube manually to apply the changes."
	unknownSourcesMsgFmt             = ":exclamation: The %s %s not found in configuration. To learn how to add custom source, visit https://botkube.io/docs/configuration/source."
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
	PersistSourceBindings(ctx context.Context, commGroupName string, platform config.CommPlatformIntegration, channelAlias string, sourceBindings []string) error
}

// EditExecutor provides functionality to run all BotKube edit related commands.
type EditExecutor struct {
	log               logrus.FieldLogger
	analyticsReporter AnalyticsReporter
	cfgManager        BindingsStorage
	sources           map[string]string
	cfg               config.Config
}

// NewEditExecutor returns a new EditExecutor instance.
func NewEditExecutor(log logrus.FieldLogger, analyticsReporter AnalyticsReporter, cfgManager BindingsStorage, cfg config.Config) *EditExecutor {
	normalizedSource := map[string]string{}
	for key, item := range cfg.Sources {
		displayName := item.DisplayName
		if displayName == "" {
			displayName = key // fallback to key
		}
		normalizedSource[key] = displayName
	}

	return &EditExecutor{
		log:               log,
		analyticsReporter: analyticsReporter,
		cfgManager:        cfgManager,
		sources:           normalizedSource,
		cfg:               cfg,
	}
}

// Do executes a given edit command based on args.
func (e *EditExecutor) Do(args []string, commGroupName string, platform config.CommPlatformIntegration, conversation Conversation, userID, botName string) (interactive.Message, error) {
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
		err := e.analyticsReporter.ReportCommand(platform, cmdToReport, conversation.IsButtonClickOrigin)
		if err != nil {
			e.log.Errorf("while reporting edit command: %s", err.Error())
		}
	}()

	cmds := executorsRunner{
		SourceBindings.Key(): func() (interactive.Message, error) {
			return e.editSourceBindingHandler(cmdArgs, commGroupName, platform, conversation, userID, botName)
		},
	}

	msg, err := cmds.SelectAndRun(cmdVerb)
	if err != nil {
		cmdVerb = anonymizedInvalidVerb // prevent passing any personal information
		return empty, err
	}
	return msg, nil
}

func (e *EditExecutor) editSourceBindingHandler(cmdArgs []string, commGroupName string, platform config.CommPlatformIntegration, conversation Conversation, userID, botName string) (interactive.Message, error) {
	var empty interactive.Message

	sourceBindings, err := e.normalizeSourceItems(cmdArgs)
	if err != nil {
		return empty, fmt.Errorf("while normalizing source args: %w", err)
	}

	if len(sourceBindings) == 0 {
		selectedOptions := e.currentlySelectedOptions(commGroupName, platform, conversation.ID)
		return interactive.Message{
			Type: interactive.Popup,
			Base: interactive.Base{
				Header: "Adjust notifications",
			},
			OnlyVisibleForYou: true,
			Sections: []interactive.Section{
				{
					MultiSelect: interactive.MultiSelect{
						Name: "Adjust notifications",
						Description: interactive.Body{
							Plaintext: "Select notification sources.",
						},
						Command:        fmt.Sprintf("%s %s", botName, "edit SourceBindings"),
						Options:        e.allOptions(),
						InitialOptions: selectedOptions,
					},
				},
			},
		}, nil
	}

	unknown := e.getUnknownInputSourceBindings(sourceBindings)
	if len(unknown) > 0 {
		return e.generateUnknownMessage(unknown), nil
	}

	err = e.cfgManager.PersistSourceBindings(context.Background(), commGroupName, platform, conversation.Alias, sourceBindings)
	if err != nil {
		return empty, fmt.Errorf("while persisting source bindings configuration: %w", err)
	}

	sourceList := english.OxfordWordSeries(e.mapToDisplayNames(sourceBindings), "and")
	if userID == "" {
		userID = "Anonymous"
	}

	return interactive.Message{
		Base: interactive.Base{
			Description: e.getEditedSourceBindingsMsg(userID, sourceList),
		},
	}, nil
}

func (e *EditExecutor) getEditedSourceBindingsMsg(userID, sourceList string) string {
	if !e.cfg.ConfigWatcher.Enabled {
		return fmt.Sprintf(editedSourcesMsgWithoutReloadFmt, userID, sourceList)
	}

	return fmt.Sprintf(editedSourcesMsgFmt, userID, sourceList)
}

func (e *EditExecutor) generateUnknownMessage(unknown []string) interactive.Message {
	list := english.OxfordWordSeries(e.quoteEachItem(unknown), "and")
	word := english.PluralWord(len(unknown), "source was", "sources were")
	return interactive.Message{
		Base: interactive.Base{
			Description: fmt.Sprintf(unknownSourcesMsgFmt, list, word),
		},
	}
}

func (e *EditExecutor) currentlySelectedOptions(commGroupName string, platform config.CommPlatformIntegration, conversationID string) []interactive.OptionItem {
	switch platform {
	case config.SlackCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Slack.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return e.mapToOptions(channel.Bindings.Sources)
		}
	case config.SocketSlackCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].SocketSlack.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return e.mapToOptions(channel.Bindings.Sources)
		}
	case config.MattermostCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Mattermost.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return e.mapToOptions(channel.Bindings.Sources)
		}
	case config.DiscordCommPlatformIntegration:
		channels := e.cfg.Communications[commGroupName].Mattermost.Channels
		for _, channel := range channels {
			if channel.Identifier() != conversationID {
				continue
			}
			return e.mapToOptions(channel.Bindings.Sources)
		}
	case config.TeamsCommPlatformIntegration:
		return e.mapToOptions(e.cfg.Communications[commGroupName].Teams.Bindings.Sources)
	}
	return nil
}

func (e *EditExecutor) mapToDisplayNames(in []string) []string {
	var out []string
	for _, key := range in {
		out = append(out, e.sources[key])
	}
	return out
}

func (e *EditExecutor) mapToOptions(in []string) []interactive.OptionItem {
	var options []interactive.OptionItem
	for _, key := range in {
		displayName, found := e.sources[key]
		if !found {
			continue
		}
		options = append(options, interactive.OptionItem{
			Name:  displayName,
			Value: key,
		})
	}
	return options
}

func (e *EditExecutor) allOptions() []interactive.OptionItem {
	var options []interactive.OptionItem
	for key, displayName := range e.sources {
		options = append(options, interactive.OptionItem{
			Name:  displayName,
			Value: key,
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Value < options[j].Value
	})

	return options
}

func (*EditExecutor) normalizeSourceItems(args []string) ([]string, error) {
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

func (e *EditExecutor) getUnknownInputSourceBindings(sources []string) []string {
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

func (*EditExecutor) quoteEachItem(in []string) []string {
	for idx := range in {
		in[idx] = fmt.Sprintf("'%s'", in[idx])
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
