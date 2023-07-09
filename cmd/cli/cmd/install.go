package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/install"
)

// NewInstall returns a cobra.Command for installing Botkube.
func NewInstall() *cobra.Command {
	var opts install.Config

	installCmd := &cobra.Command{
		Use:   "install [OPTIONS]",
		Short: "install Botkube into cluster",
		Long:  "Use this command to install the Botkube agent.",
		Example: heredoc.WithCLIName(`
			# Install latest stable Botkube version
			<cli> install

			# Install Botkube 0.1.0 version
			<cli> install --version 0.1.0

			# Install Botkube from local git repository. Needs to be run from the main directory.
			<cli> install --repo @local`, cli.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			k8sCfg, err := config.GetConfig() // fixme kubex
			if err != nil {
				return errors.Wrap(err, "while creating k8s config")
			}

			return install.Install(cmd.Context(), os.Stdout, k8sCfg, opts)
		},
	}

	flags := installCmd.Flags()

	// common params for install and upgrade operation
	flags.StringVar(&opts.HelmParams.Version, "version", install.LatestVersionTag, "Botkube version. Possible values @latest, 0.3.0, ...")
	flags.StringVar(&opts.HelmParams.Namespace, "namespace", install.Namespace, "Botkube installation namespace.")
	flags.StringVar(&opts.HelmParams.ReleaseName, "release-name", install.ReleaseName, "Botkube Helm chart release name.")
	flags.StringVar(&opts.HelmParams.RepoLocation, "repo", install.HelmRepoStable, fmt.Sprintf("Botkube Helm chart repository location. It can be relative path to current working directory or URL. Use %s tag to select repository which holds the stable Helm chart versions.", install.StableVersionTag))
	flags.BoolVar(&opts.HelmParams.DryRun, "dry-run", false, "Simulate an install")
	flags.BoolVar(&opts.HelmParams.Force, "force", false, "Force resource updates through a replacement strategy")
	flags.BoolVar(&opts.HelmParams.DisableHooks, "no-hooks", false, "Disable pre/post install/upgrade hooks")
	flags.BoolVar(&opts.HelmParams.DisableOpenAPIValidation, "disable-openapi-validation", false, "If set, it will not validate rendered templates against the Kubernetes OpenAPI Schema")
	flags.BoolVar(&opts.HelmParams.SkipCRDs, "skip-crds", false, "If set, no CRDs will be installed.")
	flags.DurationVar(&opts.HelmParams.Timeout, "timeout", 300*time.Second, "time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	flags.BoolVar(&opts.HelmParams.Wait, "wait", false, "If set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout")
	flags.BoolVar(&opts.HelmParams.WaitForJobs, "wait-for-jobs", false, "If set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout")
	flags.BoolVar(&opts.HelmParams.Atomic, "atomic", false, "If set, process rolls back changes made in case of failed install/upgrade. The --wait flag will be set automatically if --atomic is used")
	flags.BoolVar(&opts.HelmParams.SubNotes, "render-subchart-notes", false, "If set, render subchart notes along with the parent")
	flags.StringVar(&opts.HelmParams.Description, "description", "", "add a custom description")
	flags.BoolVar(&opts.HelmParams.DependencyUpdate, "dependency-update", false, "Update dependencies if they are missing before installing the chart")

	// custom values settings
	flags.StringSliceVarP(&opts.HelmParams.Values.ValueFiles, "values", "f", []string{}, "Specify values in a YAML file or a URL (can specify multiple)")
	flags.StringArrayVar(&opts.HelmParams.Values.Values, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	flags.StringArrayVar(&opts.HelmParams.Values.StringValues, "set-string", []string{}, "Set STRING values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	flags.StringArrayVar(&opts.HelmParams.Values.FileValues, "set-file", []string{}, "Set values from respective files specified via the command line (can specify multiple or separate values with commas: key1=path1,key2=path2)")
	flags.StringArrayVar(&opts.HelmParams.Values.JSONValues, "set-json", []string{}, "Set JSON values on the command line (can specify multiple or separate values with commas: key1=jsonval1,key2=jsonval2)")
	flags.StringArrayVar(&opts.HelmParams.Values.LiteralValues, "set-literal", []string{}, "Set a literal STRING value on the command line")

	// upgrade only
	flags.BoolVar(&opts.HelmParams.ReuseValues, "reuse-values", false, "When upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored")
	flags.BoolVar(&opts.HelmParams.ResetValues, "reset-values", false, "When upgrading, reset the values to the ones built into the chart")

	return installCmd
}
