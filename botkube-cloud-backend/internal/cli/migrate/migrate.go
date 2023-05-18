package migrate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hasura/go-graphql-client"
	bkconfig "github.com/kubeshop/botkube/pkg/config"
	"github.com/muesli/reflow/indent"
	"golang.org/x/oauth2"

	cliconfig "github.com/kubeshop/botkube-cloud/botkube-cloud-backend/cmd/cli/cmd/config"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/cli/printer"
	"github.com/kubeshop/botkube-cloud/botkube-cloud-backend/internal/ptr"
	gqlModel "github.com/kubeshop/botkube-cloud/botkube-cloud-backend/pkg/graphql"
)

// Run runs the migration process.
func Run(ctx context.Context, status *printer.StatusPrinter, config []byte, opts Options) (string, error) {
	cfg, err := cliconfig.New()
	if err != nil {
		return "", err
	}

	status.Step("Parsing Botkube configuration")
	botkubeClusterConfig, _, err := bkconfig.LoadWithDefaults([][]byte{config})
	if err != nil {
		return "", err
	}

	return migrate(ctx, status, opts, botkubeClusterConfig, cfg.Token)
}

func migrate(ctx context.Context, status *printer.StatusPrinter, opts Options, botkubeClusterConfig *bkconfig.Config, token string) (string, error) {
	converter := NewConverter()
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := graphql.NewClient(opts.CloudAPIURL, httpClient)

	plugins, err := converter.ConvertPlugins(botkubeClusterConfig.Executors, botkubeClusterConfig.Sources)
	if err != nil {
		return "", err
	}
	status.Step("Converted %d plugins", len(plugins))

	actions := converter.ConvertActions(botkubeClusterConfig.Actions)
	status.Step("Converted %d action", len(actions))

	platforms := converter.ConvertPlatforms(botkubeClusterConfig.Communications)
	status.Step(`Converted platforms:
    - Slacks: %d
    - Discords: %d
    - Mattermosts: %d`,
		len(platforms.SocketSlacks), len(platforms.Discords), len(platforms.Mattermosts))
	status.End(true)

	instanceName, err := getInstanceName(opts)
	if err != nil {
		return "", err
	}
	status.Step("Creating %q Cloud Instance", instanceName)
	var mutation struct {
		CreateDeployment struct {
			ID          string  `json:"id"`
			HelmCommand *string `json:"helmCommand"`
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
		return "", err
	}

	helmCmd := ptr.ToValue(mutation.CreateDeployment.HelmCommand)
	cmds := []string{
		"helm repo add botkube https://charts.botkube.io",
		"helm repo update botkube",
		helmCmd,
	}
	bldr := strings.Builder{}
	for _, cmd := range cmds {
		msg := fmt.Sprintf("$ %s\n\n", cmd)
		bldr.WriteString(indent.String(msg, 4))
	}
	status.InfoWithBody("Connect Botkube instance", bldr.String())

	run := false
	prompt := &survey.Confirm{
		Message: "Would you like to run the upgrade?",
		Default: true,
	}

	err = survey.AskOne(prompt, &run)
	if err != nil {
		return "", err
	}
	if !run {
		status.Infof("Skipping command execution. Remember to run it manually to finish the migration process.")
		return "", nil
	}

	status.Infof("Running helm upgrade")
	for _, cmd := range cmds {
		//nolint:gosec //subprocess launched with variable
		cmd := exec.Command("/bin/sh", "-c", cmd)
		cmd.Stderr = NewIndentWriter(os.Stderr, 4)
		cmd.Stdout = NewIndentWriter(os.Stdout, 4)

		if err = cmd.Run(); err != nil {
			return "", err
		}
		fmt.Println()
		fmt.Println()
	}

	status.End(true)

	return mutation.CreateDeployment.ID, nil
}

func getInstanceName(opts Options) (string, error) {
	if opts.InstanceName != "" {
		return opts.InstanceName, nil
	}

	qs := []*survey.Question{
		{
			Name: "instanceName",
			Prompt: &survey.Input{
				Message: "Please type Botkube Instance name: ",
				Default: "Botkube",
			},
			Validate: survey.ComposeValidators(survey.Required),
		},
	}

	if err := survey.Ask(qs, &opts); err != nil {
		return "", err
	}

	return opts.InstanceName, nil
}
