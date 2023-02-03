package kubectl

import (
	"fmt"
	"reflect"
	"strings"
)

type Commands struct {
	NotSupportedCommands
}

// NotSupportedCommands defines all explicitly not supported Kubectl plugin commands and their flags.
type NotSupportedCommands struct {
	Edit        *EditCommand        `arg:"subcommand:edit"`
	Attach      *AttachCommand      `arg:"subcommand:attach"`
	PortForward *PortForwardCommand `arg:"subcommand:port-forward"`
	Proxy       *ProxyCommand       `arg:"subcommand:proxy"`
	Copy        *CopyCommand        `arg:"subcommand:copy"`
	Debug       *DebugCommand       `arg:"subcommand:debug"`
	Completion  *CompletionCommand  `arg:"subcommand:completion"`
}

const tagArgName = "arg"

func returnErrorOfAllSetFlags(in any) error {
	var setFlags []string
	vv := reflect.ValueOf(in)
	fields := reflect.VisibleFields(reflect.TypeOf(in))

	for _, field := range fields {
		flagName, _ := field.Tag.Lookup(tagArgName)
		if vv.FieldByIndex(field.Index).IsZero() {
			continue
		}

		setFlags = append(setFlags, flagName)
	}

	if len(setFlags) > 0 {
		return newUnsupportedFlagsError(setFlags)
	}

	return nil
}

func newUnsupportedFlagsError(flags []string) error {
	if len(flags) == 1 {
		return fmt.Errorf("The %q flag is not supported by the Botkube Helm plugin. Please remove it.", flags[0])
	}

	points := make([]string, len(flags))
	for i, err := range flags {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Errorf(
		"Those flags are not supported by the Botkube Helm Plugin:\n\t%s\nPlease remove them.",
		strings.Join(points, "\n\t"))
}

type noopValidator struct {
	Output []string `arg:"positional"`
}

// Validate does nothing. It can be used if no validation is required,
// but you want to satisfy the command interface.
func (noopValidator) Validate() error {
	return nil
}

// EditCommand edits a resource on the server.
// It opens an editor, and we don't support that currently.
type (
	EditCommand struct {
		noopValidator
	}
	// AttachCommand attach to a running container.
	AttachCommand struct {
		noopValidator
	}
	// PortForwardCommand Forward one or more local ports to a pod
	PortForwardCommand struct {
		noopValidator
	}
	// ProxyCommand a proxy to the Kubernetes API server
	ProxyCommand struct {
		noopValidator
	}
	// CopyCommand files and directories to and from containers
	CopyCommand struct {
		noopValidator
	}
	// DebugCommand debugging sessions for troubleshooting workloads and nodes
	DebugCommand struct {
		noopValidator
	}
	// CompletionCommand shell completion code for the specified shell (bash, zsh, fish, or powershell)
	CompletionCommand struct {
		noopValidator
	}
	// ConfigCommand kubeconfig files
	ConfigCommand struct {
		noopValidator
	}
	// PluginCommand utilities for interacting with plugins.
	PluginCommand struct {
		noopValidator
	}
)
