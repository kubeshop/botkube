package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/install/helm"
	"github.com/kubeshop/botkube/internal/cli/uninstall"
	"github.com/kubeshop/botkube/internal/kubex"
)

// NewUninstall returns a cobra.Command for deleting Botkube Helm release.
func NewUninstall() *cobra.Command {
	var opts uninstall.Config

	uninstallCmd := &cobra.Command{
		Use:     "uninstall [OPTIONS]",
		Short:   "uninstall Botkube from cluster",
		Long:    "Use this command to uninstall the Botkube agent.",
		Aliases: []string{"uninstall", "del", "delete", "un"},
		Example: heredoc.WithCLIName(`
			# Uninstall default Botkube Helm release
			<cli> uninstall

			# Uninstall specific Botkube Helm release
			<cli> uninstall --release-name botkube-dev`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := kubex.LoadRestConfigWithMetaInformation()
			if err != nil {
				return err
			}
			if err != nil {
				return fmt.Errorf("while creating k8s config: %w", err)
			}

			return uninstall.Uninstall(cmd.Context(), os.Stdout, config, opts)
		},
	}

	flags := uninstallCmd.Flags()

	kubex.RegisterKubeconfigFlag(flags)

	flags.StringVar(&opts.HelmParams.ReleaseName, "release-name", helm.ReleaseName, "Botkube Helm release name.")
	flags.StringVar(&opts.HelmParams.ReleaseNamespace, "namespace", helm.Namespace, "Botkube namespace.")
	flags.BoolVarP(&opts.AutoApprove, "auto-approve", "y", false, "Skips interactive approval for deletion.")

	flags.BoolVar(&opts.HelmParams.DryRun, "dry-run", false, "simulate a uninstall")
	flags.BoolVar(&opts.HelmParams.DisableHooks, "no-hooks", false, "prevent hooks from running during uninstallation")
	flags.BoolVar(&opts.HelmParams.KeepHistory, "keep-history", false, "remove all associated resources and mark the release as deleted, but retain the release history")
	flags.BoolVar(&opts.HelmParams.Wait, "wait", true, "if set, will wait until all the resources are deleted before returning. It will wait for as long as --timeout")
	flags.StringVar(&opts.HelmParams.DeletionPropagation, "cascade", "background", "Must be \"background\", \"orphan\", or \"foreground\". Selects the deletion cascading strategy for the dependents. Defaults to background.")
	flags.DurationVar(&opts.HelmParams.Timeout, "timeout", 300*time.Second, "time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	flags.StringVar(&opts.HelmParams.Description, "description", "", "add a custom description")

	return uninstallCmd
}
