package x

import (
	"context"
	"strings"

	"github.com/gookit/color"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/executor/x/getter"
	"github.com/kubeshop/botkube/internal/executor/x/state"
	"github.com/kubeshop/botkube/internal/executor/x/template"
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

// Runner runs command and parse its output if needed.
type Runner struct {
	renderer *Renderer
	log      logrus.FieldLogger
}

// NewRunner returns a new Runner instance.
func NewRunner(log logrus.FieldLogger, renderer *Renderer) *Runner {
	return &Runner{
		log:      log,
		renderer: renderer,
	}
}

// Run runs a given command and parse its output if needed.
func (i *Runner) Run(ctx context.Context, cfg Config, state *state.Container, cmd Command, runFn func() (string, error)) (executor.ExecuteOutput, error) {
	templates, err := i.getTemplates(ctx, cfg)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	cmdTemplate, tplFound := template.FindTemplate(templates, cmd.ToExecute)

	log := i.log.WithFields(logrus.Fields{
		"isRawRequired": cmd.IsRawRequired,
		"skipExecution": cmdTemplate.SkipCommandExecution,
		"foundTemplate": tplFound,
	})

	var cmdOutput string
	if !cmdTemplate.SkipCommandExecution {
		log.WithField("command", cmd.ToExecute).Error("Running command")
		cmdOutput, err = runFn()
		if err != nil {
			return executor.ExecuteOutput{}, err
		}
	}

	if !cmd.IsRawRequired && tplFound {
		log.Info("Rendering message based on template")
		render, err := i.renderer.Get(cmdTemplate.Type)
		if err != nil {
			return executor.ExecuteOutput{}, err
		}

		cmdTemplate.TutorialMessage.Paginate.CurrentPage = cmd.PageIndex
		message, err := render.RenderMessage(cmd.ToExecute, cmdOutput, state, &cmdTemplate)
		if err != nil {
			return executor.ExecuteOutput{}, err
		}
		return executor.ExecuteOutput{
			Message: message,
		}, nil
	}

	log.Infof("Return directly got command output")
	if cmdOutput == "" {
		return executor.ExecuteOutput{}, nil // return empty message, so Botkube can convert it into "cricket sound" message
	}

	return executor.ExecuteOutput{
		Message: api.NewCodeBlockMessage(color.ClearCode(cmdOutput), true),
	}, nil
}

func (i *Runner) getTemplates(ctx context.Context, cfg Config) ([]template.Template, error) {
	templates, err := getter.Load[template.Template](ctx, cfg.TmpDir.GetDirectory(), cfg.Templates)
	if err != nil {
		return nil, err
	}

	for _, tpl := range templates {
		i.log.WithFields(logrus.Fields{
			"trigger": tpl.Trigger.Command,
			"type":    tpl.Type,
		}).Debug("Command template")
	}
	return templates, nil
}

// RunInstalledCommand runs a given user command for already installed CLIs.
func RunInstalledCommand(ctx context.Context, tmp plugin.TmpDir, in string, envs map[string]string) (string, error) {
	opts := []pluginx.ExecuteCommandMutation{
		pluginx.ExecuteCommandEnvs(envs),
	}

	path, custom := tmp.Get()
	if custom {
		// we installed all assets in different directory, e.g. because we run it locally,
		// so we override the default deps path
		opts = append(opts, pluginx.ExecuteCommandDependencyDir(path))
	}

	out, err := pluginx.ExecuteCommand(ctx, in, opts...)
	if err != nil {
		return "", err
	}

	var str strings.Builder
	str.WriteString(color.ClearCode(out.Stdout))
	if out.Stderr != "" {
		str.WriteString("\n")
		str.WriteString(color.ClearCode(out.Stderr))
	}
	return strings.TrimSpace(str.String()), nil
}
