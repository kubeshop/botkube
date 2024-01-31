package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	pluginName       = "gh"
	logsTailLines    = 150
	defaultNamespace = "default"
	helpMsg          = "Usage: `gh create issue KIND/NAME [-n, --namespace]`"
	description      = "GH creates an issue on GitHub for a related Kubernetes resource."
)

var (
	// version is set via ldflags by GoReleaser.
	version = "dev"

	//go:embed config_schema.json
	configJSONSchema string
)

// Config holds the GitHub executor configuration.
type Config struct {
	GitHub struct {
		Token         string `yaml:"token"`
		Repository    string `yaml:"repository"`
		IssueTemplate string `yaml:"issueTemplate"`
	}
	Log config.Logger `yaml:"log"`
}

// Commands defines all supported GitHub plugin commands and their flags.
type (
	Commands struct {
		Create *CreateCommand `arg:"subcommand:create"`
	}
	CreateCommand struct {
		Issue *CreateIssueCommand `arg:"subcommand:issue"`
	}
	CreateIssueCommand struct {
		Type      string `arg:"positional"`
		Namespace string `arg:"-n,--namespace"`
	}
)

// GHExecutor implements the Botkube executor plugin interface.
type GHExecutor struct{}

// Metadata returns details about the GitHub plugin.
func (*GHExecutor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:          version,
		Description:      description,
		Dependencies:     depsDownloadLinks,
		DocumentationURL: "https://botkube.io/blog/build-a-github-issues-reporter-for-failing-kubernetes-apps-with-botkube-plugins",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
	}, nil
}

// Execute returns a given command as a response.
func (e *GHExecutor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	if err := pluginx.ValidateKubeConfigProvided(pluginName, in.Context.KubeConfig); err != nil {
		return executor.ExecuteOutput{}, err
	}

	var cfg Config
	err := pluginx.MergeExecutorConfigs(in.Configs, &cfg)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}

	var cmd Commands
	err = pluginx.ParseCommand(pluginName, in.Command, &cmd)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while parsing input command: %w", err)
	}

	if cmd.Create == nil || cmd.Create.Issue == nil {
		return executor.ExecuteOutput{
			Message: api.NewCodeBlockMessage(fmt.Sprintf("Usage: %s create issue KIND/NAME", pluginName), false),
		}, nil
	}

	log := loggerx.New(cfg.Log)

	kubeConfigPath, deleteFn, err := pluginx.PersistKubeConfig(ctx, in.Context.KubeConfig)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while writing kubeconfig file: %w", err)
	}
	defer func() {
		if deleteErr := deleteFn(ctx); deleteErr != nil {
			log.Errorf("failed to delete kubeconfig file %s: %w", kubeConfigPath, deleteErr)
		}
	}()

	issueDetails, err := getIssueDetails(ctx, cmd.Create.Issue.Namespace, cmd.Create.Issue.Type, kubeConfigPath)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while fetching logs : %w", err)
	}

	mdBody, err := renderIssueBody(cfg.GitHub.IssueTemplate, issueDetails)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while rendering issue body: %w", err)
	}

	title := fmt.Sprintf("The `%s` malfunctions", cmd.Create.Issue.Type)
	issueURL, err := createGitHubIssue(cfg, title, mdBody)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while creating GitHub issue: %w", err)
	}

	return executor.ExecuteOutput{
		Message: api.NewCodeBlockMessage(fmt.Sprintf("New issue created successfully! 🎉\n\nIssue URL: %s", issueURL), false),
	}, nil
}

// Help returns help message
func (*GHExecutor) Help(context.Context) (api.Message, error) {
	return api.NewPlaintextMessage(helpMsg, true), nil
}

var depsDownloadLinks = map[string]api.Dependency{
	// Links source: https://github.com/cli/cli/releases/tag/v2.29.0
	"gh": {
		URLs: map[string]string{
			// Using go-getter syntax to unwrap the underlying directory structure.
			// Read more on https://github.com/hashicorp/go-getter#subdirectories
			"darwin/amd64": "https://github.com/cli/cli/releases/download/v2.29.0/gh_2.29.0_macOS_amd64.zip//gh_2.29.0_macOS_amd64/bin",
			"darwin/arm64": "https://github.com/cli/cli/releases/download/v2.29.0/gh_2.29.0_macOS_arm64.zip//gh_2.29.0_macOS_arm64/bin",
			"linux/amd64":  "https://github.com/cli/cli/releases/download/v2.29.0/gh_2.29.0_linux_amd64.zip//gh_2.29.0_linux_amd64/bin",
			"linux/arm64":  "https://github.com/cli/cli/releases/download/v2.29.0/gh_2.29.0_linux_arm64.zip//gh_2.29.0_linux_arm64/bin",
			"linux/386":    "https://github.com/cli/cli/releases/download/v2.29.0/gh_2.29.0_linux_386.zip//gh_2.29.0_linux_386/bin",
		},
	},
	"kubectl": {
		URLs: map[string]string{
			"darwin/amd64": "https://dl.k8s.io/release/v1.26.0/bin/darwin/amd64/kubectl",
			"darwin/arm64": "https://dl.k8s.io/release/v1.26.0/bin/darwin/arm64/kubectl",
			"linux/amd64":  "https://dl.k8s.io/release/v1.26.0/bin/linux/amd64/kubectl",
			"linux/arm64":  "https://dl.k8s.io/release/v1.26.0/bin/linux/arm64/kubectl",
			"linux/386":    "https://dl.k8s.io/release/v1.26.0/bin/linux/386/kubectl",
		},
	},
}

func main() {
	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: &GHExecutor{},
		},
	})
}

func createGitHubIssue(cfg Config, title, mdBody string) (string, error) {
	cmd := fmt.Sprintf("gh issue create --title %q --body '%s' --label bug -R %s", title, mdBody, cfg.GitHub.Repository)

	envs := map[string]string{
		"GH_TOKEN": cfg.GitHub.Token,
	}

	output, err := pluginx.ExecuteCommand(context.Background(), cmd, pluginx.ExecuteCommandEnvs(envs))
	if err != nil {
		return "", err
	}
	return output.Stdout, nil
}

// IssueDetails holds all available information about a given issue.
type IssueDetails struct {
	Type      string
	Namespace string
	Logs      string
	Version   string
}

func getIssueDetails(ctx context.Context, namespace, name, kubeConfigPath string) (IssueDetails, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	logs, err := pluginx.ExecuteCommand(ctx, fmt.Sprintf("kubectl --kubeconfig=%s logs %s -n %s --tail %d", kubeConfigPath, name, namespace, logsTailLines))
	if err != nil {
		return IssueDetails{}, fmt.Errorf("while getting logs: %w", err)
	}
	ver, err := pluginx.ExecuteCommand(ctx, fmt.Sprintf("kubectl --kubeconfig=%s version -o yaml", kubeConfigPath))
	if err != nil {
		return IssueDetails{}, fmt.Errorf("while getting version: %w", err)
	}

	return IssueDetails{
		Type:      name,
		Namespace: namespace,
		Logs:      logs.Stdout,
		Version:   ver.Stdout,
	}, nil
}

const defaultIssueBody = `
## Description

This issue refers to the problems connected with {{ .Type | code "bash" }} in namespace {{ .Namespace | code "bash" }}

<details>
  <summary><b>Logs</b></summary>
{{ .Logs | code "bash"}}
</details>

### Cluster details

{{ .Version | code "yaml"}}
`

func renderIssueBody(bodyTpl string, data IssueDetails) (string, error) {
	if bodyTpl == "" {
		bodyTpl = defaultIssueBody
	}

	tmpl, err := template.New("issue-body").Funcs(template.FuncMap{
		"code": func(syntax, in string) string {
			return fmt.Sprintf("\n```%s\n%s\n```\n", syntax, in)
		},
	}).Parse(bodyTpl)
	if err != nil {
		return "", fmt.Errorf("while creating template: %w", err)
	}

	var body bytes.Buffer
	err = tmpl.Execute(&body, data)
	if err != nil {
		return "", fmt.Errorf("while generating body: %w", err)
	}

	return body.String(), nil
}
