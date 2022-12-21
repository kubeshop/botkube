package execute

import (
	"bytes"
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const (
	helpMsgHeader = "%s %s [feature]\n\nAvailable features:\n"
	// noFeature is used for commands that have no features defined
	noFeature = ""
	// incompleteCmdMsg incomplete command response message
	incompleteCmdMsg = "You missed to pass options for the command. Please use 'help' to see command options."
)

// CommandVerb are commands supported by the bot
type CommandVerb string

// CommandVerb command options
const (
	CommandPing     CommandVerb = "ping"
	CommandHelp     CommandVerb = "help"
	CommandVersion  CommandVerb = "version"
	CommandFeedback CommandVerb = "feedback"
	CommandList     CommandVerb = "list"
	CommandEnable   CommandVerb = "enable"
	CommandDisable  CommandVerb = "disable"
	CommandEdit     CommandVerb = "edit"
	CommandStart    CommandVerb = "start"
	CommandStop     CommandVerb = "stop"
	CommandStatus   CommandVerb = "status"
	CommandConfig   CommandVerb = "config"
)

// CommandExecutor defines command structure for executors
type CommandExecutor interface {
	Commands() map[CommandVerb]CommandFn
	FeatureName() FeatureName
}

// CommandFn is a single command (eg. List())
type CommandFn func(ctx context.Context, cmdCtx CommandContext) (interactive.Message, error)

// CommandContext contains the context for CommandFn
type CommandContext struct {
	Args                []string
	ClusterName         string
	CommGroupName       string
	BotName             string
	RawCmd              string
	CleanCmd            string
	ProvidedClusterName string
	User                string
	Conversation        Conversation
	Platform            config.CommPlatformIntegration
	ExecutorFilter      executorFilter
	NotifierHandler     NotifierHandler
	Mapping             *CommandMapping
}

// ProvidedClusterNameEqualOrEmpty returns true when provided cluster name is empty
// or when provided cluster name is equal to cluster name
func (cmdCtx CommandContext) ProvidedClusterNameEqualOrEmpty() bool {
	return cmdCtx.ProvidedClusterName == "" || cmdCtx.ProvidedClusterNameEqual()
}

// ProvidedClusterNameEqual returns true when provided cluster name is equal to cluster name
func (cmdCtx CommandContext) ProvidedClusterNameEqual() bool {
	return cmdCtx.ProvidedClusterName == cmdCtx.ClusterName
}

// FeatureName defines the name and aliases for a feature
type FeatureName struct {
	Name    string
	Aliases []string
}

// CommandMapping allows to register and lookup commands and dynamically build help messages
type CommandMapping struct {
	commands map[CommandVerb]map[string]CommandFn
	help     map[CommandVerb][]FeatureName
}

// NewCmdsMapping registers command and help mappings
func NewCmdsMapping(executors []CommandExecutor) (*CommandMapping, error) {
	mappingsErrs := multierror.New()
	cmdsMapping := make(map[CommandVerb]map[string]CommandFn)
	helpMapping := make(map[CommandVerb][]FeatureName)
	for _, executor := range executors {
		cmds := executor.Commands()
		subCmd := executor.FeatureName()
		for verb, cmdFn := range cmds {
			if value := cmdsMapping[verb]; value == nil {
				cmdsMapping[verb] = make(map[string]CommandFn)
			}
			if value := helpMapping[verb]; value == nil {
				helpMapping[verb] = make([]FeatureName, 0)
			}
			cmdsMapping[verb][subCmd.Name] = cmdFn
			helpMapping[verb] = append(helpMapping[verb], subCmd)
			for _, featureName := range subCmd.Aliases {
				if _, ok := cmdsMapping[verb][featureName]; ok {
					mappingsErrs = multierror.Append(mappingsErrs, fmt.Errorf("command collision: tried to register '%s %s', but it already exists", verb, featureName))
				}
				cmdsMapping[verb][featureName] = cmdFn
			}
		}
	}
	if err := mappingsErrs.ErrorOrNil(); err != nil {
		return nil, err
	}
	return &CommandMapping{
		commands: cmdsMapping,
		help:     helpMapping,
	}, nil
}

// FindFn looks up CommandFn by verb and feature
func (m *CommandMapping) FindFn(verb CommandVerb, feature string) (CommandFn, bool, bool) {
	features, ok := m.commands[verb]
	if !ok {
		return nil, false, false
	}
	fn, ok := features[feature]
	if !ok {
		return nil, true, false
	}
	return fn, true, true
}

// HelpMessageForVerb dynamically builds help message for given CommandVerb, or empty string
func (m *CommandMapping) HelpMessageForVerb(verb CommandVerb, botName string) string {
	cmd, ok := m.help[verb]
	if !ok {
		return incompleteCmdMsg
	}
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 3, 0, 1, ' ', 0)

	fmt.Fprintf(w, helpMsgHeader, botName, verb)
	for _, feature := range cmd {
		aliases := removeEmptyFeatures(feature.Aliases)
		fmtStr := fmt.Sprintf("%s\t", feature.Name)
		for _, a := range aliases {
			fmtStr += fmt.Sprintf("|\t%s\t", a)
		}
		fmt.Fprintln(w, fmtStr)
	}
	w.Flush()
	return buf.String()
}

func removeEmptyFeatures(features []string) []string {
	clean := make([]string, 0, len(features))
	for _, f := range features {
		if f != "" {
			clean = append(clean, f)
		}
	}
	return clean
}
