package doctor

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/PullRequestInc/go-gpt3"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const (
	PluginName        = "doctor"
	promptTemplate    = "Can you show me 3 possible kubectl commands to take an action after resource '%s' in namespace '%s' (if namespace needed) fails with error '%s'?"
	defaultAPIBaseURL = "https://api.openai.com/v1"
	defaultUserAgent  = "go-gpt3"
)

var (
	//go:embed config-jsonschema.json
	configJSONSchema string
	k8sPromptRegex   = regexp.MustCompile(`--(\w+)=([^\s]+)`)
)

type Config struct {
	APIBaseURL     string `yaml:"apiBaseUrl"`
	APIKey         string `yaml:"apiKey"`
	DefaultEngine  string `yaml:"defaultEngine"`
	OrganizationID string `yaml:"organizationID"`
	UserAgent      string `yaml:"userAgent"`
}

// Executor provides functionality for running Doctor.
type Executor struct {
	pluginVersion string
	gptClient     gpt3.Client
	l             sync.Mutex
}

// NewExecutor returns a new Executor instance.
func NewExecutor(ver string) *Executor {
	return &Executor{
		pluginVersion: ver,
	}
}

// Metadata returns details about the Doctor plugin.
func (d *Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     d.pluginVersion,
		Description: "Doctor is a ChatGPT integration project that knows how to diagnose Kubernetes problems and suggest solutions.",
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
	}, nil
}

// Execute returns a given command as a response.
func (d *Executor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	var cfg Config
	err := pluginx.MergeExecutorConfigs(in.Configs, &cfg)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configuration: %w", err)
	}
	doctorParams, err := normalizeCommand(in.Command)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while normalizing command: %w", err)
	}
	gpt, err := d.getGptClient(&cfg)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while initializing GPT client: %w", err)
	}
	sb := strings.Builder{}
	err = gpt.CompletionStream(ctx,
		gpt3.CompletionRequest{
			Prompt:      []string{buildPrompt(doctorParams)},
			MaxTokens:   gpt3.IntPtr(300),
			Temperature: gpt3.Float32Ptr(0),
		}, func(resp *gpt3.CompletionResponse) {
			text := resp.Choices[0].Text
			sb.WriteString(text)
		})
	if err != nil {
		return executor.ExecuteOutput{}, err
	}
	response := sb.String()
	response = strings.TrimLeft(response, "\n")
	if doctorParams.IsRaw() {
		return executor.ExecuteOutput{
			Message: api.NewPlaintextMessage(response, true),
		}, nil
	}
	btnBuilder := api.NewMessageButtonBuilder()
	var btns []api.Button
	for i, s := range strings.Split(response, "\n") {
		parts := strings.Split(s, "")
		if len(parts) < 4 {
			continue
		}
		s = strings.Join(parts[3:], "")
		btns = append(btns, btnBuilder.ForCommandWithDescCmd(fmt.Sprintf("Choice %d", i+1), s, api.ButtonStylePrimary))
	}
	return executor.ExecuteOutput{
		Message: api.Message{
			BaseBody: api.Body{
				Plaintext: "Possible actions",
			},
			Sections: []api.Section{
				{
					Buttons: btns,
				},
			},
			OnlyVisibleForYou: false,
			ReplaceOriginal:   false,
		},
	}, nil
}

// Help returns help message
func (d *Executor) Help(context.Context) (api.Message, error) {
	btnBuilder := api.NewMessageButtonBuilder()
	return api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header:      "Run `doctor` commands",
					Description: "Doctor helps in finding the root cause of a k8s problem.",
				},
				Buttons: []api.Button{
					btnBuilder.ForCommandWithDescCmd("Run", "doctor 'text'"),
				},
			},
		},
	}, nil
}

func (d *Executor) getGptClient(cfg *Config) (gpt3.Client, error) {
	d.l.Lock()
	defer d.l.Unlock()

	if d.gptClient != nil {
		return d.gptClient, nil
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API Key cannot be empty. If you use OpenAI API, generate it here: https://platform.openai.com/account/api-keys")
	}

	baseURL := defaultAPIBaseURL
	if cfg.APIBaseURL != "" {
		baseURL = cfg.APIBaseURL
	}

	defaultEngine := gpt3.TextDavinci003Engine
	if cfg.DefaultEngine != "" {
		defaultEngine = cfg.DefaultEngine
	}

	userAgent := defaultUserAgent
	if cfg.UserAgent != "" {
		userAgent = cfg.UserAgent
	}

	orgID := ""
	if cfg.OrganizationID != "" {
		orgID = cfg.OrganizationID
	}

	opts := []gpt3.ClientOption{
		gpt3.WithBaseURL(baseURL),
		gpt3.WithDefaultEngine(defaultEngine),
		gpt3.WithUserAgent(userAgent),
		gpt3.WithOrg(orgID),
	}

	d.gptClient = gpt3.NewClient(cfg.APIKey, opts...)
	return d.gptClient, nil
}

type DoctorParams struct {
	RawText   string
	Resource  string
	Namespace string
	Error     string
}

func (p *DoctorParams) IsRaw() bool {
	return p.Resource == "" || p.Error == ""
}
func normalizeCommand(command string) (DoctorParams, error) {
	matches := k8sPromptRegex.FindAllStringSubmatch(command, -1)
	params := DoctorParams{}
	params.RawText = command
	for _, match := range matches {
		if len(match) != 3 {
			return DoctorParams{}, errors.New("invalid command")
		}
		key := match[1]
		value := match[2]

		switch key {
		case "resource":
			params.Resource = value
		case "namespace":
			params.Namespace = value
		case "error":
			params.Error = value
		}
	}
	return params, nil
}

func buildPrompt(p DoctorParams) string {
	if p.IsRaw() {
		return p.RawText
	}
	return fmt.Sprintf(promptTemplate, p.Resource, p.Namespace, p.Error)
}
