package migrate

import (
	"context"
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hasura/go-graphql-client"
	"github.com/mattn/go-shellwords"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"helm.sh/helm/v3/pkg/cli/values"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/install"
	"github.com/kubeshop/botkube/internal/cli/install/helm"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
	bkconfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const (
	defaultInstanceName = "Botkube"

	instanceDetailsURLFmt = "%s/instances/%s"
	platformNameOther     = "Other"
)

// Run runs the migration process.
func Run(ctx context.Context, status *printer.StatusPrinter, config []byte, k8sCfg *kubex.ConfigWithMeta, opts Options) (string, error) {
	authToken := opts.Token
	if authToken == "" {
		cfg, err := cli.ReadConfig()
		if err != nil {
			return "", err
		}
		authToken = cfg.Token
	}

	status.Step("Parsing Botkube configuration")
	botkubeClusterConfig, _, err := bkconfig.LoadWithDefaults([][]byte{config})
	if err != nil {
		return "", err
	}

	fmt.Println(">>> InCluster Plugins")
	formatx.StructDumper().Dump(botkubeClusterConfig.Plugins)

	return migrate(ctx, status, opts, botkubeClusterConfig, k8sCfg, authToken)
}

func migrate(ctx context.Context, status *printer.StatusPrinter, opts Options, botkubeClusterConfig *bkconfig.Config, k8sCfg *kubex.ConfigWithMeta, token string) (string, error) {
	converter := NewConverter()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := graphql.NewClient(opts.CloudAPIURL, httpClient)

	plugins, err := converter.ConvertPlugins(botkubeClusterConfig.Executors, botkubeClusterConfig.Sources)
	if err != nil {
		return "", fmt.Errorf("while converting plugins: %w", err)
	}

	pluginsCount := 0
	if len(plugins) != 0 && len(plugins[0].Groups) != 0 {
		pluginsCount = len(plugins[0].Groups)
	}
	status.Step("Converted %d plugins", pluginsCount)

	actions := converter.ConvertActions(botkubeClusterConfig.Actions, botkubeClusterConfig.Sources, botkubeClusterConfig.Executors)
	status.Step("Converted %d actions", len(actions))

	platforms := converter.ConvertPlatforms(botkubeClusterConfig.Communications)
	status.Step(`Converted platforms:
    - Slacks: %d
    - Discords: %d
    - Mattermosts: %d`,
		len(platforms.SocketSlacks), len(platforms.Discords), len(platforms.Mattermosts))
	status.End(true)

	instanceName, err := getInstanceName(opts)
	if err != nil {
		return "", fmt.Errorf("while parsing instance name: %w", err)
	}
	status.Step("Creating %q Cloud Instance", instanceName)
	var mutation struct {
		CreateDeployment struct {
			ID                         string                                            `json:"id"`
			InstallUpgradeInstructions []*gqlModel.InstallUpgradeInstructionsForPlatform `json:"installUpgradeInstructions"`
		} `graphql:"createDeployment(input: $input)"`
	}
	err = client.Mutate(ctx, &mutation, map[string]interface{}{
		"input": gqlModel.DeploymentCreateInput{
			Name:      instanceName,
			Plugins:   plugins,
			Actions:   actions,
			Platforms: platforms,
		},
	})
	if err != nil {
		return "", fmt.Errorf("while creating deployment: %w", err)
	}

	aliases := converter.ConvertAliases(botkubeClusterConfig.Aliases, mutation.CreateDeployment.ID)
	status.Step("Converted %d aliases", len(aliases))

	errs := multierror.New()
	for _, alias := range aliases {
		status.Step("Migrating Alias %q", alias.Name)
		var aliasMutation struct {
			CreateAlias struct {
				ID string `json:"id"`
			} `graphql:"createAlias(input: $input)"`
		}
		err = client.Mutate(ctx, &aliasMutation, map[string]interface{}{
			"input": *alias,
		})
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while creating Alias %q: %w", alias.Name, err))
			continue
		}
	}
	status.End(true)

	if errs.ErrorOrNil() != nil {
		return "", fmt.Errorf("while migrating aliases: %w%s", errs.ErrorOrNil(), errStateMessage(opts.CloudDashboardURL, mutation.CreateDeployment.ID))
	}

	if opts.SkipConnect {
		status.End(true)
		return mutation.CreateDeployment.ID, nil
	}

	params, err := parseHelmCommand(mutation.CreateDeployment.InstallUpgradeInstructions, opts)
	if err != nil {
		return "", fmt.Errorf("while parsing helm command: %w", err)
	}
	installConfig := install.Config{
		HelmParams: params,
		Watch:      opts.Watch,
		Timeout:    opts.Timeout,
	}
	if err := install.Install(ctx, os.Stdout, k8sCfg, installConfig); err != nil {
		return "", fmt.Errorf("while installing Botkube: %w", err)
	}

	return mutation.CreateDeployment.ID, nil
}

func getInstanceName(opts Options) (string, error) {
	if opts.InstanceName != "" {
		return opts.InstanceName, nil
	}

	if opts.AutoApprove {
		return defaultInstanceName, nil
	}

	qs := []*survey.Question{
		{
			Name: "instanceName",
			Prompt: &survey.Input{
				Message: "Please type Botkube Instance name: ",
				Default: defaultInstanceName,
			},
			Validate: survey.ComposeValidators(survey.Required),
		},
	}

	if err := survey.Ask(qs, &opts); err != nil {
		return "", err
	}

	return opts.InstanceName, nil
}

func parseHelmCommand(instructions []*gqlModel.InstallUpgradeInstructionsForPlatform, opts Options) (helm.Config, error) {
	var raw string
	for _, i := range instructions {
		if i.PlatformName == platformNameOther {
			raw = i.InstallUpgradeCommand
		}
	}
	tokenized, err := shellwords.Parse(raw)
	if err != nil {
		return helm.Config{}, fmt.Errorf("while tokenizing helm command: %w", err)
	}

	var version string
	var vals []string
	flagSet := flag.NewFlagSet("helm cmd", flag.ExitOnError)
	flagSet.StringVar(&version, "version", "", "")
	flagSet.StringArrayVar(&vals, "set", []string{}, "")
	if err := flagSet.Parse(tokenized); err != nil {
		return helm.Config{}, fmt.Errorf("while registering flags: %w", err)
	}

	if opts.ImageTag != "" {
		vals = append(vals, fmt.Sprintf("image.tag=%s", opts.ImageTag))
	}

	return helm.Config{
		Version: version,
		Values: values.Options{
			Values: vals,
		},
		Namespace:    helm.Namespace,
		ReleaseName:  helm.ReleaseName,
		ChartName:    helm.HelmChartName,
		RepoLocation: helm.HelmRepoStable,
		AutoApprove:  opts.AutoApprove,
	}, nil
}

func errStateMessage(dashboardURL, instanceID string) string {
	return fmt.Sprintf("\n\nMigration process failed. Navigate to %s to continue configuring newly created instance.\n"+
		"Alternatively, delete the instance from the link above and try again.", fmt.Sprintf(instanceDetailsURLFmt, dashboardURL, instanceID))
}
