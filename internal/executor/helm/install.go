package helm

import (
	"context"
	"io"
	"log"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
)

// Supported options:
// 1. By absolute URL:
//      helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
// 2. By chart reference and repo url:
//      helm install --repo https://example.com/charts/ mynginx nginx

func runInstall(ctx context.Context, args []string, client *action.Install, valueOpts *values.Options, out io.Writer) (*release.Release, error) {
	var settings = cli.New()

	log.Printf("Original chart version: %q", client.Version)
	if client.Version == "" && client.Devel {
		log.Printf("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	name, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, err
	}
	client.ReleaseName = name

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}

	log.Printf("CHART PATH: %s\n", cp)

	p := getter.All(settings)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}

	if chartRequested.Metadata.Deprecated {
		log.Println("This chart is deprecated") // warn
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			err = errors.Wrap(err, "An error occurred while checking for chart dependencies. You may need to run `helm dependency build` to fetch missing dependencies")
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              out,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	return client.RunWithContext(ctx, chartRequested, vals)
}

func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func newInstallClient(installArgs *InstallCmd, actionConfig *action.Configuration) *action.Install {
	client := action.NewInstall(actionConfig)
	client.CreateNamespace = installArgs.CreateNamespace
	client.DryRun = installArgs.DryRun
	client.DisableHooks = installArgs.NoHooks
	client.Replace = installArgs.Replace
	client.Wait = installArgs.Wait
	client.WaitForJobs = installArgs.WaitForJobs
	client.Devel = installArgs.Devel
	client.DependencyUpdate = installArgs.DependencyUpdate
	client.Timeout = installArgs.Timeout
	//client.Namespace =
	client.GenerateName = installArgs.GenerateName
	client.NameTemplate = installArgs.NameTemplate
	client.Description = installArgs.DescriptionD
	client.OutputDir = installArgs.Output
	client.Atomic = installArgs.Atomic
	client.SkipCRDs = installArgs.SkipCRDs
	client.DisableOpenAPIValidation = installArgs.DisableOpenAPIValidation

	client.CaFile = installArgs.CaFile
	client.CertFile = installArgs.CertFile
	client.KeyFile = installArgs.KeyFile
	client.InsecureSkipTLSverify = installArgs.InsecureSkipTLSVerify
	client.Keyring = installArgs.Keyring
	client.Password = installArgs.Password
	client.PassCredentialsAll = installArgs.PassCredentials
	client.RepoURL = installArgs.Repo
	client.Username = installArgs.Username
	client.Verify = installArgs.Verify
	client.Version = installArgs.Version

	return client
}
