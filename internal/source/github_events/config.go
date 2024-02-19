package github_events

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kubeshop/botkube/internal/source/github_events/gh"
	"github.com/kubeshop/botkube/internal/source/github_events/templates"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/plugin"
)

type (
	// Config represents the main configuration.
	Config struct {
		Log config.Logger `yaml:"log"`

		// GitHub configuration.
		GitHub gh.ClientConfig `yaml:"github"`

		// RefreshDuration defines how often we should call GitHub REST API to check repository events.
		// It's the same for all configured repositories. For example, if you configure 5s refresh time, and you have 3 repositories registered,
		// we will execute maximum 2160 calls which easily fits into PAT rate limits.
		// You can create multiple plugins configuration with dedicated tokens to have the rate limits increased.
		//
		// NOTE:
		// - we use conditional requests (https://docs.github.com/en/rest/overview/resources-in-the-rest-api?apiVersion=2022-11-28#conditional-requests), so if there are no events the call doesn't count against your rate limits.
		// - if you configure file pattern matcher for merged pull request events we execute one more additional call to check which files were changed in the context of a given pull request
		//
		// Rate limiting: https://docs.github.com/en/rest/overview/resources-in-the-rest-api?apiVersion=2022-11-28#rate-limiting
		RefreshDuration time.Duration `yaml:"refreshDuration"`

		// List of repository configurations.
		Repositories []RepositoryConfig `yaml:"repositories"`
	}

	// EventsAPIMatcher defines matchers for /events API.
	EventsAPIMatcher struct {
		// Type defines event type.
		// Required.
		Type string `yaml:"type"`

		// The JSONPath expression to filter events
		JSONPath string `yaml:"jsonPath"`

		// The value to match in the JSONPath result
		Value string `yaml:"value"`

		// NotificationTemplate defines custom notification template.
		NotificationTemplate NotificationTemplate `yaml:"notificationTemplate,omitempty"`
	}

	// FilePatterns represents the file patterns configuration.
	FilePatterns struct {
		// Includes file patterns.
		Include []string `yaml:"include"`

		// Excludes file patterns.
		Exclude []string `yaml:"exclude"`
	}

	// ExtraButton represents the extra button configuration in notification templates.
	ExtraButton struct {
		// DisplayName for the extra button.
		DisplayName string `yaml:"displayName"`

		// CommandTpl template for the extra button.
		CommandTpl string `yaml:"commandTpl"`

		// URL to open. If specified CommandTpl is ignored.
		URL string `yaml:"url"`

		// Style for button.
		Style string `yaml:"style"`
	}

	// NotificationTemplate represents the notification template configuration.
	NotificationTemplate struct {
		// Extra buttons in the notification template.
		ExtraButtons []ExtraButton `yaml:"extraButtons"`
		PreviewTpl   string        `yaml:"previewTpl"`
	}

	// RepositoryConfig represents the configuration for repositories.
	RepositoryConfig struct {
		// Repository name represents the GitHub repository.
		// It is in form 'owner/repository'.
		Name string `yaml:"name"`

		// OnMatchers defines allowed GitHub matcher criteria.
		OnMatchers On `yaml:"on"`

		// BeforeDuration is the duration used to decrease the initial time after used for filtering old events.
		// It is particularly useful during testing, allowing you to set it to 48h to retrieve events older than the plugin's start time.
		// If not specified, the plugin's start time is used as the initial value.
		BeforeDuration time.Duration `yaml:"beforeDuration"`
	}

	// On defines allowed GitHub matcher criteria.
	On struct {
		PullRequests []PullRequest `yaml:"pullRequests"`
		// EventsAPI watches for /events API
		EventsAPI []EventsAPIMatcher `yaml:"events,omitempty"`
	}
	PullRequest struct {
		// Types patterns defines if we should watch only for pull requests with given state criteria.
		// Allowed values: open, closed, merged.
		Types []string `yaml:"types,omitempty"`
		// Paths patterns defines if we should watch only for pull requests with given files criteria.
		Paths IncludeExcludeRegex `yaml:"paths,omitempty"`
		// Labels patterns define if we should watch only for pull requests with given labels criteria.
		Labels IncludeExcludeRegex `yaml:"labels,omitempty"`
		// NotificationTemplate defines custom notification template.
		NotificationTemplate NotificationTemplate `yaml:"notificationTemplate,omitempty"`
	}

	// IncludeExcludeRegex defines regex filter criteria.
	IncludeExcludeRegex struct {
		Include []string `yaml:"include"`
		Exclude []string `yaml:"exclude"`
	}
)

func (t NotificationTemplate) ToOptions() []templates.MessageMutatorOption {
	var out []templates.MessageMutatorOption
	if t.PreviewTpl != "" {
		out = append(out, WithCustomPreview(t.PreviewTpl))
	}
	if len(t.ExtraButtons) > 0 {
		out = append(out, WithExtraButtons(t.ExtraButtons))
	}

	return out
}

func (r *IncludeExcludeRegex) IsEmpty() bool {
	return len(r.Exclude) == 0 && len(r.Include) == 0
}

// IsDefined checks if a given value is defined by a given file pattern matcher.
// Firstly, it checks if the value is excluded. If not, then it checks if the value is included.
func (r *IncludeExcludeRegex) IsDefined(value string) (bool, error) {
	if r == nil {
		return false, nil
	}

	// 1. Check if excluded
	if len(r.Exclude) > 0 {
		for _, excludeValue := range r.Exclude {
			if strings.TrimSpace(excludeValue) == "" {
				continue
			}
			// exact match
			if excludeValue == value {
				return false, nil
			}

			// regexp
			matched, err := regexp.MatchString(excludeValue, value)
			if err != nil {
				return false, fmt.Errorf("while matching %q with exclude regex %q: %v", value, excludeValue, err)
			}
			if matched {
				return false, nil
			}
		}
	}

	// 2. Check if included, if matched, return true
	if len(r.Include) > 0 {
		for _, includeValue := range r.Include {
			// exact match
			if includeValue == value {
				return true, nil
			}

			// regexp
			matched, err := regexp.MatchString(includeValue, value)
			if err != nil {
				return false, fmt.Errorf("while matching %q with include regex %q: %v", value, includeValue, err)
			}
			if matched {
				return true, nil
			}
		}
	}

	// 2.1. If not included, return false
	return false, nil
}

// MergeConfigs merges all input configuration.
func MergeConfigs(configs []*source.Config) (Config, error) {
	defaults := Config{
		Log: config.Logger{
			Level: "info",
		},
		RefreshDuration: 5 * time.Second,
		GitHub: gh.ClientConfig{
			BaseURL:   "https://api.github.com/",
			UploadURL: "https://uploads.github.com/",
		},
	}
	var out Config
	if err := plugin.MergeSourceConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, err
	}

	return out, nil
}
