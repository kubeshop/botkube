package main

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/hashicorp/go-plugin"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	pluginName       = "gh"
	logsTailLines    = 150
	defaultNamespace = "default"
	helpMsg          = "Usage: `gh create issue KIND/NAME [-n, --namespace]`"
)

// version is set via ldflags by GoReleaser.
var version = "dev"

// Config holds the GitHub executor configuration.
type Config struct {
	GitHub struct {
		Token         string
		Repository    string
		IssueTemplate string
	}
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
		Version:      version,
		Description:  "GH creates an issue on GitHub for a related Kubernetes resource.",
		Dependencies: depsDownloadLinks,
	}, nil
}

// Execute returns a given command as a response.
func (e *GHExecutor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
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
			Data: fmt.Sprintf("Usage: %s create issue KIND/NAME", pluginName),
		}, nil
	}

	issueDetails, err := getIssueDetails(ctx, cmd.Create.Issue.Namespace, cmd.Create.Issue.Type)
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
		Data: fmt.Sprintf("New issue created successfully! 🎉\n\nIssue URL: %s", issueURL),
	}, nil
}

// Help returns help message
func (*GHExecutor) Help(context.Context) (api.Message, error) {
	return api.NewPlaintextMessage(helpMsg, true), nil
}

var depsDownloadLinks = map[string]api.Dependency{
	// Links source: https://github.com/cli/cli/releases/tag/v2.22.1
	"gh": {
		URLs: map[string]string{
			// Using go-getter syntax to unwrap the underlying directory structure.
			// Read more on https://github.com/hashicorp/go-getter#subdirectories
			"darwin/amd64": "https://github.com/cli/cli/releases/download/v2.22.1/gh_2.22.1_macOS_amd64.tar.gz//gh_2.22.1_macOS_amd64/bin",
			"linux/amd64":  "https://github.com/cli/cli/releases/download/v2.22.1/gh_2.22.1_linux_amd64.tar.gz//gh_2.22.1_linux_amd64/bin",
			"linux/arm64":  "https://github.com/cli/cli/releases/download/v2.22.1/gh_2.22.1_linux_arm64.tar.gz//gh_2.22.1_linux_arm64/bin",
			"linux/386":    "https://github.com/cli/cli/releases/download/v2.22.1/gh_2.22.1_linux_386.tar.gz//gh_2.22.1_linux_386/bin",
		},
	},
	"kubectl": {
		URLs: map[string]string{
			"darwin/amd64": "https://dl.k8s.io/release/v1.26.0/bin/darwin/amd64/kubectl",
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

	return pluginx.ExecuteCommandWithEnvs(context.Background(), cmd, envs)
}

// IssueDetails holds all available information about a given issue.
type IssueDetails struct {
	Type      string
	Namespace string
	Logs      string
	Version   string
}

func getIssueDetails(ctx context.Context, namespace, name string) (IssueDetails, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}
	logs, err := pluginx.ExecuteCommand(ctx, fmt.Sprintf("kubectl logs %s -n %s --tail %d", name, namespace, logsTailLines))
	if err != nil {
		return IssueDetails{}, fmt.Errorf("while getting logs: %w", err)
	}
	ver, err := pluginx.ExecuteCommand(ctx, "kubectl version -o yaml")
	if err != nil {
		return IssueDetails{}, fmt.Errorf("while getting version: %w", err)
	}

	return IssueDetails{
		Type:      name,
		Namespace: namespace,
		Logs:      logs,
		Version:   ver,
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
