package helm

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
	"helm.sh/helm/v3/pkg/action"

	"github.com/kubeshop/botkube/internal/cli/helmx"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
)

// Helm provides option to or delete Helm release.
type Helm struct {
	helmCfg *action.Configuration
}

// NewHelm returns a new Helm instance.
func NewHelm(k8sCfg *kubex.ConfigWithMeta, forNamespace string) (*Helm, error) {
	configuration, err := helmx.GetActionConfiguration(k8sCfg, forNamespace)
	if err != nil {
		return nil, err
	}
	return &Helm{helmCfg: configuration}, nil
}

// Uninstall uninstalls a given Helm release.
func (c *Helm) Uninstall(ctx context.Context, status *printer.StatusPrinter, opts Config) error {
	status.Step("Uninstalling...")
	uninstall := c.uninstallAction(opts)
	//  We may run into in issue temporary network issues.
	return retry.Do(func() error {
		if ctx.Err() != nil {
			return ctx.Err() // context cancelled or timed out.
		}

		_, err := uninstall.Run(opts.ReleaseName)
		return err
	}, retry.Attempts(3), retry.Delay(time.Second))
}

func (c *Helm) uninstallAction(opts Config) *action.Uninstall {
	deleteAction := action.NewUninstall(c.helmCfg)

	deleteAction.DisableHooks = opts.DisableHooks
	deleteAction.DryRun = opts.DryRun
	deleteAction.KeepHistory = opts.KeepHistory
	deleteAction.Wait = opts.Wait
	deleteAction.DeletionPropagation = opts.DeletionPropagation
	deleteAction.Timeout = opts.Timeout
	deleteAction.Description = opts.Description

	return deleteAction
}
