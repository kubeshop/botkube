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
func (i *Runner) Run(ctx context.Context, cfg Config, state *state.Container, tool string, kubeconfigPath string) (executor.ExecuteOutput, error) {
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

	out, err := runCmd(ctx, cfg.TmpDir, cmd.ToExecute, map[string]string{
		"KUBECONFIG": kubeconfigPath,
	})
	if err != nil {
		i.log.WithError(err).WithField("command", cmd.ToExecute).Error("failed to run command")
		return executor.ExecuteOutput{}, err
	}

	if cmd.IsRawRequired {
		i.log.Info("Raw output was explicitly requested")
		return executor.ExecuteOutput{
			Message: api.NewCodeBlockMessage(out, true),
		}, nil
	}

	cmdTemplate, found := template.FindWithPrefix(templates, cmd.ToExecute)
	if !found {
		i.log.Info("Templates config not found for command")
		return executor.ExecuteOutput{
			Message: api.NewCodeBlockMessage(color.ClearCode(out), true),
		}, nil
	}

	render, err := i.renderer.Get(cmdTemplate.Type)
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

func runCmd(ctx context.Context, tmp plugin.TmpDir, in string, envs map[string]string) (string, error) {
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
