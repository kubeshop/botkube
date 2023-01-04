package helm

import (
	"errors"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/mattn/go-shellwords"
)

// Commands defines all supported Helm plugin commands and their flags.
type Commands struct {
	Install  *InstallCommand  `arg:"subcommand:install"`
	Version  *VersionCommand  `arg:"subcommand:version"`
	Status   *StatusCommand   `arg:"subcommand:status"`
	Test     *TestCommand     `arg:"subcommand:test"`
	Rollback *RollbackCommand `arg:"subcommand:rollback"`
	Upgrade  *UpgradeCommand  `arg:"subcommand:upgrade"`
	Help     *HelpCommand     `arg:"subcommand:help"`
	Get      *GetCommand      `arg:"subcommand:get"`

	// embed on the root of the Command struct to inline all aliases.
	HistoryCommandAliases
	UninstallCommandAliases
	ListCommandAliases

	GlobalFlags
}

// GlobalFlags holds flags supported by all Helm plugin commands
type GlobalFlags struct {
	Namespace  string `arg:"--namespace,-n"`
	Debug      bool   `arg:"--debug"`
	BurstLimit int    `arg:"--burst-limit"`
}

func parseRawCommand(rawCmd string) (Commands, []string, error) {
	rawCmd = strings.TrimSpace(rawCmd)
	if !strings.HasPrefix(rawCmd, PluginName) {
		return Commands{}, nil, errors.New("the input command does not target the Helm plugin executor")
	}
	rawCmd = strings.TrimPrefix(rawCmd, PluginName)

	var helmCmd Commands
	p, err := arg.NewParser(arg.Config{}, &helmCmd)
	if err != nil {
		return helmCmd, nil, err
	}

	args, err := shellwords.Parse(rawCmd)
	if err != nil {
		return helmCmd, nil, err
	}
	err = p.Parse(removeVersionFlag(args))
	if err != nil {
		return helmCmd, nil, err
	}

	return helmCmd, args, nil
}

// The go-arg library is handling the `--version` flag internally, see:
// https://github.com/alexflint/go-arg/blob/727f8533acca70ca429dce4bfea729a6af75c3f7/parse.go#L610
//
// In case of Helm the `--version` flag has a different purpose, so we just remove it for now.
func removeVersionFlag(args []string) []string {
	for idx := range args {
		if !strings.HasPrefix(args[idx], "--version") {
			continue
		}
		prev := idx
		next := idx + 1

		if !strings.Contains(args[idx], "=") { // val is in next arg: --version 1.2.3
			next = next + 1
		}

		if next > len(args) {
			next = len(args)
		}
		return append(args[:prev], args[next:]...)
	}
	return args
}

type noopValidator struct{}

// Validate does nothing. It can be used if no validation is required,
// but you want to satisfy the command interface.
func (noopValidator) Validate() error {
	return nil
}
