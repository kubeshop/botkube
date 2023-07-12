package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/alexflint/go-arg"
	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/executor/x"
	"github.com/kubeshop/botkube/internal/executor/x/getter"
	"github.com/kubeshop/botkube/internal/executor/x/output"
	"github.com/kubeshop/botkube/internal/executor/x/state"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

const pluginName = "exec"

// XExecutor implements Botkube executor plugin.
type XExecutor struct{}

func (i *XExecutor) Help(_ context.Context) (api.Message, error) {
	help := heredoc.Doc(`
		Usage:
		  exec run [COMMAND] [FLAGS]    Run a specified command with optional flags
		  exec install [SOURCE]         Install a binary using the https://github.com/zyedidia/eget syntax.
		
		Usage Examples:
		  # Install the Helm CLI

		  exec install https://get.helm.sh/helm-v3.10.3-linux-amd64.tar.gz --file helm    
		  
		  # Run the 'helm list -A' command.

		  exec run helm list -A    
		
		Options:
		  -h, --help                 Show this help message`)
	return api.NewCodeBlockMessage(help, true), nil
}

// Metadata returns details about Echo plugin.
func (*XExecutor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:      version,
		Description:  "Install and run CLIs directly from chat window without hassle. All magic included.",
		Dependencies: x.GetPluginDependencies(),
		JSONSchema:   jsonSchema(),
	}, nil
}

type (
	Commands struct {
		Install *InstallCmd `arg:"subcommand:install"`
		Run     *RunCmd     `arg:"subcommand:run"`
	}
	InstallCmd struct {
		Tool []string `arg:"positional"`
	}
	RunCmd struct {
		Tool []string `arg:"positional"`
	}
)

func escapePositionals(in string) string {
	for _, name := range []string{"run", "install"} {
		if strings.Contains(in, name) {
			return strings.Replace(in, name, fmt.Sprintf("%s -- ", name), 1)
		}
	}
	return in
}

// Execute returns a given command as response.
//
//nolint:gocritic // hugeParam: in is heavy (80 bytes); consider passing it by pointer
func (i *XExecutor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	var cmd Commands
	in.Command = escapePositionals(in.Command)
	err := pluginx.ParseCommand(pluginName, in.Command, &cmd)
	switch err {
	case nil:
	case arg.ErrHelp:
		msg, _ := i.Help(ctx)
		return executor.ExecuteOutput{
			Message: msg,
		}, nil
	default:
		return executor.ExecuteOutput{}, fmt.Errorf("while parsing input command: %w", err)
	}

	cfg := x.Config{
		Templates: []getter.Source{
			{Ref: getDefaultTemplateSource()},
		},
	}
	if err := pluginx.MergeExecutorConfigs(in.Configs, &cfg); err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging configs: %v", err)
	}

	log := loggerx.New(cfg.Logger)

	renderer := x.NewRenderer()
	err = renderer.RegisterAll(map[string]x.Render{
		"parser:table:.*": output.NewTableCommandParser(log),
		"wrapper":         output.NewCommandWrapper(),
		"tutorial":        output.NewTutorialWrapper(),
	})
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while registering message renderers: %v", err)
	}

	runner := x.NewRunner(log, renderer)

	state := state.ExtractSlackState(in.Context.SlackState)

	switch {
	case cmd.Run != nil:
		tool := Normalize(strings.Join(cmd.Run.Tool, " "))
		log.WithField("tool", tool).Info("Running command...")

		command := x.Parse(tool)
		run := func() (string, error) {
			kubeConfigPath, deleteFn, err := i.getKubeconfig(ctx, log, in)
			defer deleteFn()
			if err != nil {
				return "", fmt.Errorf("while creating kubeconfig: %v", err)
			}

			out, err := x.RunInstalledCommand(ctx, cfg.TmpDir, command.ToExecute, map[string]string{
				"KUBECONFIG": kubeConfigPath,
			})
			if err != nil {
				log.WithError(err).WithField("command", command.ToExecute).Error("failed to run command")
				return "", fmt.Errorf("while running command: %v", err)
			}
			return out, nil
		}

		return runner.Run(ctx, cfg, state, command, run)
	case cmd.Install != nil:
		var (
			tool          = Normalize(strings.Join(cmd.Install.Tool, " "))
			command       = x.Parse(fmt.Sprintf("exec install %s", tool))
			dir, isCustom = cfg.TmpDir.Get()
			downloadCmd   = fmt.Sprintf("eget %s", tool)
		)

		run := func() (string, error) {
			log.WithFields(logrus.Fields{
				"dir":         dir,
				"isCustom":    isCustom,
				"userCommand": command.ToExecute,
				"runCommand":  downloadCmd,
			}).Info("Installing binary...")

			if _, err := pluginx.ExecuteCommand(ctx, downloadCmd, pluginx.ExecuteCommandEnvs(map[string]string{
				"EGET_BIN": dir,
			})); err != nil {
				return "", err
			}

			return "Binary was installed successfully 🎉", nil
		}

		return runner.Run(ctx, cfg, state, command, run)
	}
	return executor.ExecuteOutput{
		Message: api.NewPlaintextMessage("Command not supported", false),
	}, nil
}

func (i *XExecutor) getKubeconfig(ctx context.Context, log logrus.FieldLogger, in executor.ExecuteInput) (string, func(), error) {
	if len(in.Context.KubeConfig) == 0 {
		return "", func() {}, nil
	}
	kubeConfigPath, deleteFn, err := pluginx.PersistKubeConfig(ctx, in.Context.KubeConfig)
	if err != nil {
		return "", func() {}, fmt.Errorf("while writing kubeconfig file: %w", err)
	}

	return kubeConfigPath, func() {
		err := deleteFn(ctx)
		if err != nil {
			log.WithError(err).WithField("kubeconfigPath", kubeConfigPath).Error("Failed to delete kubeconfig file")
		}
	}, nil
}

func main() {
	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: &XExecutor{},
		},
	})
}

// jsonSchema returns JSON schema for the executor.
func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`{
		  "$schema": "http://json-schema.org/draft-07/schema#",
		  "title": "exec",
		  "description": "Install and run CLIs directly from the chat window without hassle. All magic included.",
		  "type": "object",
		  "properties": {
			"templates": {
			  "type": "array",
			  "title": "List of templates",
			  "description": "An array of templates that define how to convert the command output into an interactive message.",
			  "items": {
				"type": "object",
				"properties": {
				  "ref": {
					"title": "Link to templates source",
					"description": "It uses the go-getter library, which supports multiple URL formats (such as HTTP, Git repositories, or S3) and is able to unpack archives. For more details, see the documentation at https://github.com/hashicorp/go-getter.",
					"type": "string",
					"default": "%s"
				  }
				},
				"required": [
				  "ref"
				],
				"additionalProperties": false
			  }
			}
		  },
		  "required": [
			"templates"
		  ]
		}`, getDefaultTemplateSource()),
	}
}

func getDefaultTemplateSource() string {
	ver := version
	if ver == "dev" {
		ver = "main"
	}
	return fmt.Sprintf("github.com/kubeshop/botkube//cmd/executor/exec/templates?ref=%s", ver)
}

func Normalize(in string) string {
	out := formatx.RemoveHyperlinks(in)
	out = strings.NewReplacer(`“`, `"`, `”`, `"`, `‘`, `"`, `’`, `"`).Replace(out)

	out = strings.TrimSpace(out)

	return out
}
