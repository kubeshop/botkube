package helm

import (
	"errors"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/mattn/go-shellwords"
)

// Commands defines all supported Helm plugin subcommands and their flags.
type Commands struct {
	Install *InstallCommand `arg:"subcommand:install"`
	// Global Helm plugin plugins
	Namespace  string `arg:"--namespace,-n"`
	Debug      bool   `arg:"--debug"`
	BurstLimit int    `arg:"--burst-limit"`
}

func parseRawCommand(rawCmd string) (Commands, []string, error) {
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
	err = p.Parse(args)
	if err != nil {
		return helmCmd, nil, err
	}

	return helmCmd, args, nil
}
