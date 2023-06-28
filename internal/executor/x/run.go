package x

import (
	"context"
	"os"

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

type Runner struct {
	renderer *Renderer
	log      logrus.FieldLogger
}

func NewRunner(log logrus.FieldLogger, renderer *Renderer) *Runner {
	return &Runner{
		log:      log,
		renderer: renderer,
	}
}

func (i *Runner) Run(ctx context.Context, cfg Config, state *state.Container, tool string) (executor.ExecuteOutput, error) {
	cmd := Parse(tool)

	templates, err := getter.Load[template.Template](ctx, cfg.TmpDir.GetDirectory(), cfg.Templates)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	for _, tpl := range templates {
		i.log.WithFields(logrus.Fields{
			"trigger": tpl.Trigger.Command,
			"type":    tpl.Type,
		}).Info("Command template")
	}

	cmdTemplate, found := template.FindWithPrefix(templates, cmd.ToExecute)

	var out string
	if !found || cmdTemplate.Type != "tutorial" {
		out, err = runCmd(ctx, cfg.TmpDir, cmd.ToExecute)
		if err != nil {
			i.log.WithError(err).WithField("command", cmd.ToExecute).Error("failed to run command")
			return executor.ExecuteOutput{}, err
		}
	}

	if cmd.IsRawRequired {
		i.log.Info("Raw output was explicitly requested")
		return executor.ExecuteOutput{
			Message: api.NewCodeBlockMessage(out, true),
		}, nil
	}

	if !found {
		i.log.Info("Templates config not found for command")
		return executor.ExecuteOutput{
			Message: api.NewCodeBlockMessage(color.ClearCode(out), true),
		}, nil
	}

	render, err := i.renderer.Get(cmdTemplate.Type) // Message.Type
	if err != nil {
		return executor.ExecuteOutput{}, err
	}

	message, err := render.RenderMessage(cmd.ToExecute, out, state, &cmdTemplate)
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	return executor.ExecuteOutput{
		Message: message,
	}, nil
}

func runCmd(ctx context.Context, tmp plugin.TmpDir, in string) (string, error) {
	path, custom := tmp.Get()
	if custom {
		// FIXME: get rid of it
		defer os.Setenv("PLUGIN_DEPENDENCY_DIR", os.Getenv("PLUGIN_DEPENDENCY_DIR"))
		os.Setenv("PLUGIN_DEPENDENCY_DIR", path)
	}

	out, err := pluginx.ExecuteCommand(ctx, in)
	if err != nil {
		return "", err
	}
	return color.ClearCode(out), nil
}
