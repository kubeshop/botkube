package migrate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hasura/go-graphql-client"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"helm.sh/helm/v3/pkg/cli/values"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/cli"
	cliconfig "github.com/kubeshop/botkube/internal/cli/config"
	"github.com/kubeshop/botkube/internal/cli/install"
	"github.com/kubeshop/botkube/internal/cli/install/helm"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
	gqlModel "github.com/kubeshop/botkube/internal/remote/graphql"
	bkconfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const (
	migrationJobName    = "botkube-migration"
	configMapName       = "botkube-config-exporter"
	defaultInstanceName = "Botkube"

	instanceDetailsURLFmt = "%s/instances/%s"
	platformNameOther     = "Other"
)

// Run runs the migration process.
func Run(ctx context.Context, status *printer.StatusPrinter, config []byte, k8sCfg *kubex.ConfigWithMeta, opts Options) (string, error) {
	authToken := opts.Token
	if authToken == "" {
		cfg, err := cliconfig.New()
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
		return "", errors.Wrap(err, "while converting plugins")
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
		return "", errors.Wrap(err, "while parsing instance name")
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
		return "", errors.Wrap(err, "while creating deployment")
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
		return "", errors.Wrap(err, "while parsing helm command")
	}
	installConfig := install.Config{
		HelmParams: params,
		Watch:      opts.Watch,
		Timeout:    opts.Timeout,
	}
	if err := install.Install(ctx, os.Stdout, k8sCfg, installConfig); err != nil {
		return "", errors.Wrap(err, "while installing Botkube")
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

func GetConfigFromCluster(ctx context.Context, k8sCfg *rest.Config, opts Options) ([]byte, *corev1.Pod, error) {
	k8sCli, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return nil, nil, err
	}
	defer cleanup(ctx, k8sCli, opts)

	botkubePod, err := getBotkubePod(ctx, k8sCli, opts)
	if err != nil {
		return nil, nil, err
	}

	if err = createMigrationJob(ctx, k8sCli, botkubePod, opts.ConfigExporter); err != nil {
		return nil, nil, err
	}

	if err = waitForMigrationJob(ctx, k8sCli, opts); err != nil {
		return nil, nil, err
	}
	config, err := readConfigFromCM(ctx, k8sCli, opts)
	if err != nil {
		return nil, nil, err
	}
	return config, botkubePod, nil
}

func getBotkubePod(ctx context.Context, k8sCli *kubernetes.Clientset, opts Options) (*corev1.Pod, error) {
	pods, err := k8sCli.CoreV1().Pods(opts.Namespace).List(ctx, metav1.ListOptions{LabelSelector: opts.Label})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no botkube pod found")
	}
	return &pods.Items[0], nil
}

func createMigrationJob(ctx context.Context, k8sCli *kubernetes.Clientset, botkubePod *corev1.Pod, cfg ConfigExporterOptions) error {
	var container corev1.Container
	for _, c := range botkubePod.Spec.Containers {
		if c.Name == "botkube" {
			container = c
			break
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      migrationJobName,
			Namespace: botkubePod.Namespace,
			Labels: map[string]string{
				"app":                  migrationJobName,
				"botkube.io/migration": "true",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            migrationJobName,
							Image:           fmt.Sprintf("%s/%s:%s", cfg.Registry, cfg.Repository, cfg.Tag),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             container.Env,
							VolumeMounts:    container.VolumeMounts,
						},
					},
					Volumes:            botkubePod.Spec.Volumes,
					ServiceAccountName: botkubePod.Spec.ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
				},
			},
		},
	}

	_, err := k8sCli.BatchV1().Jobs(botkubePod.Namespace).Create(ctx, job, metav1.CreateOptions{})

	return err
}

func waitForMigrationJob(ctx context.Context, k8sCli *kubernetes.Clientset, opts Options) error {
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, opts.ConfigExporter.Timeout)
	defer cancelFn()

	ticker := time.NewTicker(opts.ConfigExporter.PollPeriod)
	defer ticker.Stop()

	var job *batchv1.Job
	for {
		select {
		case <-ctxWithTimeout.Done():

			errMsg := fmt.Sprintf("migration job failed: %s", context.Canceled.Error())

			if cli.VerboseMode.IsEnabled() && job != nil {
				errMsg = fmt.Sprintf("%s\n\nDEBUG:\njob:\n\n%s", errMsg, job.String())
			}

			// TODO: Add ability to keep the job if it fails and improve the error
			return errors.New(errMsg)
		case <-ticker.C:
			var err error
			job, err = k8sCli.BatchV1().Jobs(opts.Namespace).Get(ctx, migrationJobName, metav1.GetOptions{})
			if err != nil {
				fmt.Println("Error getting migration job: ", err.Error())
				continue
			}

			if job.Status.Succeeded > 0 {
				return nil
			}
		}
	}
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
		return helm.Config{}, errors.Wrap(err, "could not tokenize helm command")
	}

	var version string
	var vals []string
	flagSet := flag.NewFlagSet("helm cmd", flag.ExitOnError)
	flagSet.StringVar(&version, "version", "", "")
	flagSet.StringArrayVar(&vals, "set", []string{}, "")
	if err := flagSet.Parse(tokenized); err != nil {
		return helm.Config{}, errors.Wrap(err, "could not register flags")
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

func readConfigFromCM(ctx context.Context, k8sCli *kubernetes.Clientset, opts Options) ([]byte, error) {
	configMap, err := k8sCli.CoreV1().ConfigMaps(opts.Namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap.BinaryData["config.yaml"], nil
}

func cleanup(ctx context.Context, k8sCli *kubernetes.Clientset, opts Options) {
	foreground := metav1.DeletePropagationForeground
	_ = k8sCli.BatchV1().Jobs(opts.Namespace).Delete(ctx, migrationJobName, metav1.DeleteOptions{PropagationPolicy: &foreground})
	_ = k8sCli.CoreV1().ConfigMaps(opts.Namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
}

func errStateMessage(dashboardURL, instanceID string) string {
	return fmt.Sprintf("\n\nMigration process failed. Navigate to %s to continue configuring newly created instance.\n"+
		"Alternatively, delete the instance from the link above and try again.", fmt.Sprintf(instanceDetailsURLFmt, dashboardURL, instanceID))
}
