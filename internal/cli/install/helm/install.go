package helm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/avast/retry-go/v4"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/helmx"
	"github.com/kubeshop/botkube/internal/cli/install/iox"
	"github.com/kubeshop/botkube/internal/cli/printer"
)

const restartAnnotationFmt = "extraAnnotations.cli\\.botkube\\.io\\/restart\\-timestamp=\"%d\""

// Run provides single function signature both for install and upgrade.
type Run func(ctx context.Context, relName string, chart *chart.Chart, vals map[string]any) (*release.Release, error)

// Helm provides option to or update install Helm charts.
type Helm struct {
	helmCfg *action.Configuration
}

// NewHelm returns a new Helm instance.
func NewHelm(k8sCfg *rest.Config, forNamespace string) (*Helm, error) {
	configuration, err := helmx.GetActionConfiguration(k8sCfg, forNamespace)
	if err != nil {
		return nil, err
	}
	return &Helm{helmCfg: configuration}, nil
}

// Install installs a given Helm chart.
func (c *Helm) Install(ctx context.Context, status *printer.StatusPrinter, opts Config) (*release.Release, error) {
	histClient := action.NewHistory(c.helmCfg)
	rels, err := histClient.Run(opts.ReleaseName)
	var runFn Run
	switch {
	case err == nil:
		if len(rels) > 0 { // it shouldn't happen, because there is not found error in such cases, however it better to be on the safe side.
			if err := PrintReleaseStatus("Detected existing Botkube installation:", status, rels[len(rels)-1]); err != nil {
				return nil, err
			}
		} else {
			status.Infof("Detected existing Botkube installation")
		}

		switch opts.AutoApprove {
		case true:
			status.Infof("Upgrade process will proceed as auto-approval has been explicitly specified")
		case false:
			prompt := &survey.Confirm{
				Message: "Do you want to upgrade existing installation?",
				Default: true,
			}

			var upgrade bool

			questionIndent := iox.NewIndentStdoutWriter("?", 1) // we indent questions by 1 space to match the step layout
			err = survey.AskOne(prompt, &upgrade, survey.WithStdio(os.Stdin, questionIndent, os.Stderr))
			if err != nil {
				return nil, fmt.Errorf("while confiriming upgrade: %v", err)
			}

			if !upgrade {
				return nil, errors.New("upgrade aborted")
			}
		}
		restartAnnotation := fmt.Sprintf(restartAnnotationFmt, time.Now().Unix())
		if cli.VerboseMode.IsEnabled() {
			status.Step("Appending %q Pod annotation to enforce Pod restart", restartAnnotation)
		} else {
			status.Step("Appending Pod annotation to enforce Pod restart")
		}
		opts.Values.Values = append(opts.Values.Values, restartAnnotation)
		status.End(true)

		runFn = c.upgradeAction(opts)
	case err == driver.ErrReleaseNotFound:
		runFn = c.installAction(opts)
	default:
		return nil, fmt.Errorf("while getting Helm release history: %v", err)
	}

	status.Step("Loading %s Helm chart", opts.ChartName)
	loadedChart, cleanup, err := c.getChart(opts.RepoLocation, opts.ChartName, opts.Version)
	if err != nil {
		return nil, fmt.Errorf("while loading Helm chart: %v", err)
	}
	defer cleanup()

	p := getter.All(helmcli.New())
	vals, err := opts.Values.MergeValues(p)
	if err != nil {
		return nil, err
	}

	status.Step("Scheduling %s Helm chart", opts.ChartName)
	status.End(true)
	//  We may run into in issue temporary network issues.
	var rel *release.Release
	err = retry.Do(func() error {
		rel, err = runFn(ctx, opts.ReleaseName, loadedChart, vals)
		return err
	}, retry.Attempts(3), retry.Delay(time.Second))
	if err != nil {
		return nil, err
	}

	return rel, nil
}

func (c *Helm) getChart(repoLocation string, chartName string, version string) (*chart.Chart, func(), error) {
	location := chartName
	chartOptions := action.ChartPathOptions{
		RepoURL: repoLocation,
		Version: version,
	}

	if isLocalDir(repoLocation) {
		repoLocation = strings.TrimSuffix(repoLocation, "/")
		location = fmt.Sprintf("%s/%s", repoLocation, chartName)
		chartOptions.RepoURL = ""
	}

	temp, err := os.MkdirTemp("", "botkube-helm-repo")
	if err != nil {
		return nil, nil, err
	}

	chartPath, err := chartOptions.LocateChart(location, &helmcli.EnvSettings{
		RepositoryCache: temp,
	})
	if err != nil {
		return nil, nil, err
	}

	chartData, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, err
	}

	return chartData, func() {
		_ = os.RemoveAll(temp) // it will be anyway garbage collected by OS after some time.
	}, nil
}

func (c *Helm) installAction(opts Config) Run {
	installCli := action.NewInstall(c.helmCfg)

	installCli.Namespace = opts.Namespace
	installCli.SkipCRDs = opts.SkipCRDs
	installCli.Wait = false // botkube CLI has a custom logic to do that
	installCli.WaitForJobs = false
	installCli.DisableHooks = opts.DisableHooks
	installCli.DryRun = opts.DryRun
	installCli.Force = opts.Force

	installCli.Atomic = opts.Atomic
	installCli.SubNotes = opts.SubNotes
	installCli.Description = opts.Description
	installCli.DisableOpenAPIValidation = opts.DisableOpenAPIValidation
	installCli.DependencyUpdate = opts.DependencyUpdate

	return func(ctx context.Context, relName string, chart *chart.Chart, vals map[string]any) (*release.Release, error) {
		installCli.ReleaseName = relName
		return installCli.RunWithContext(ctx, chart, vals)
	}
}

func (c *Helm) upgradeAction(opts Config) Run {
	upgradeAction := action.NewUpgrade(c.helmCfg)

	upgradeAction.Namespace = opts.Namespace
	upgradeAction.SkipCRDs = opts.SkipCRDs
	upgradeAction.Wait = false // botkube CLI has a custom logic to do that
	upgradeAction.WaitForJobs = false
	upgradeAction.DisableHooks = opts.DisableHooks
	upgradeAction.DryRun = opts.DryRun
	upgradeAction.Force = opts.Force
	upgradeAction.ResetValues = opts.ResetValues
	upgradeAction.ReuseValues = opts.ReuseValues
	upgradeAction.Atomic = opts.Atomic
	upgradeAction.SubNotes = opts.SubNotes
	upgradeAction.Description = opts.Description
	upgradeAction.DisableOpenAPIValidation = opts.DisableOpenAPIValidation
	upgradeAction.DependencyUpdate = opts.DependencyUpdate

	return func(ctx context.Context, relName string, chart *chart.Chart, vals map[string]any) (*release.Release, error) {
		return upgradeAction.RunWithContext(ctx, relName, chart, vals)
	}
}

func isLocalDir(in string) bool {
	f, err := os.Stat(in)
	return err == nil && f.IsDir()
}
