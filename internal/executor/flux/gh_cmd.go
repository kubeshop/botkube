package flux

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

type (
	GitHubCommand struct {
		Command []string `arg:"positional"`
	}
)

type PRDetails struct {
	Author Author `json:"author,omitempty"`
	State  string `json:"state,omitempty"`
	URL    string `json:"url,omitempty"`
}
type Author struct {
	ID    string `json:"id,omitempty"`
	IsBot bool   `json:"is_bot,omitempty"`
	Login string `json:"login,omitempty"`
	Name  string `json:"name,omitempty"`
}

type GitHubCmdService struct {
	log logrus.FieldLogger
}

func NewGitHubCmdService(log logrus.FieldLogger) *GitHubCmdService {
	return &GitHubCmdService{
		log: log,
	}
}

func (k *GitHubCmdService) ShouldHandle(command string) (*GitHubCommand, bool) {
	if !strings.Contains(command, "gh") {
		return nil, false
	}

	command = escapePositionals(command, "gh")
	var gh struct {
		GitHub *GitHubCommand `arg:"subcommand:gh"`
	}

	err := pluginx.ParseCommand(PluginName, command, &gh)
	if err != nil {
		// if we cannot parse, it means that unknown command was specified
		k.log.WithError(err).Debug("Cannot parse input command into gh ones.")
		return nil, false
	}

	if gh.GitHub == nil {
		return nil, false
	}
	return gh.GitHub, true
}

func (k *GitHubCmdService) Run(ctx context.Context, diffCmd *GitHubCommand, cfg Config, opts ...pluginx.ExecuteCommandMutation) (executor.ExecuteOutput, error) {
	cmdToRun := "gh " + strings.Join(diffCmd.Command, " ")

	opts = append(opts, pluginx.ExecuteCommandEnvs(map[string]string{
		"GH_TOKEN": cfg.GitHub.Auth.AccessToken,
	}))

	out, err := ExecuteCommand(ctx, cmdToRun, opts...)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while running command: %v", err)
	}

	if strings.TrimSpace(out) == "" {
		// gh CLI detects non-interactive TTY and in some cases doesn't return any message.
		return executor.ExecuteOutput{
			Message: api.Message{
				BaseBody: api.Body{
					Plaintext: "Command successfully executed! 🎉",
				},
			},
		}, nil
	}

	return executor.ExecuteOutput{
		Message: api.Message{
			BaseBody: api.Body{
				CodeBlock: out,
			},
		},
	}, nil
}
