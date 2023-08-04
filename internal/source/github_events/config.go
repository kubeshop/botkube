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
	"github.com/kubeshop/botkube/pkg/pluginx"
)

type (
	// Config represents the main configuration.
	Config struct {
		Log config.Logger `yaml:"log"`

		// GitHub configuration.
		GitHub gh.ClientConfig `yaml:"github"`

		// RefreshTime defines how often we should call GitHub REST API to check repository events.
		// It's the same for all configured repositories. For example, if you configure 5s refresh time, and you have 3 repositories registered,
		// we will execute maximum 2160 calls which easily fits into PAT rate limits. You need to consider when you configure this plugin.
		// You can create multiple plugins configuration with dedicated tokens to split have the rate limits increased.
		//
		// NOTE:
		// - we use conditional requests (https://docs.github.com/en/rest/overview/resources-in-the-rest-api?apiVersion=2022-11-28#conditional-requests), so if there are no events the call doesn't count against your rate limits.
		// - if you configure file pattern matcher for merged pull request events we execute one more additional call to check which files were changed in the context of a given pull request
		//
		// Rate limiting: https://docs.github.com/en/rest/overview/resources-in-the-rest-api?apiVersion=2022-11-28#rate-limiting
		//
		// Defaults: 2s
		RefreshTime time.Duration `yaml:"refreshTime"`

		// List of repository configurations.
		Repositories []RepositoryConfig `yaml:"repositories"`
	}

	// JSONPathMatcher represents the JSONPath matcher configuration.
	JSONPathMatcher struct {
		// The JSONPath expression to filter events
		JSONPath string `yaml:"jsonPath"`

		// The value to match in the JSONPath result
		Value string `yaml:"value"`
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
		// Display name for the extra button.
		DisplayName string `yaml:"displayName"`

		// Command template for the extra button.
		CommandTpl string `yaml:"commandTpl"`
		Style      string `yaml:"style"`
	}

	// NotificationTemplate represents the notification template configuration.
	NotificationTemplate struct {
		// Extra buttons in the notification template.
		ExtraButtons []ExtraButton `yaml:"extraButtons"`
	}

	// RepositoryConfig represents the configuration for repositories.
	RepositoryConfig struct {
		// Repository name represents the GitHub repository.
		// It is in form 'owner/repository'.
		Name string `yaml:"name"`

		// OnMatchers defines allowed GitHub matcher criteria.
		OnMatchers On `yaml:"on"`
	}

	// On defines allowed GitHub matcher criteria.
	On struct {
		PullRequests []PullRequest `yaml:"pullRequests,omitempty"`
		// JSON path matcher configuration.
		//JSONPathMatcher []JSONPathMatcher `yaml:"jsonPathMatcher"`
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
		RefreshTime: 5 * time.Second,
		GitHub: gh.ClientConfig{
			BaseURL:   "https://api.github.com/",
			UploadURL: "https://uploads.github.com/",
		},
	}
	var out Config
	if err := pluginx.MergeSourceConfigsWithDefaults(defaults, configs, &out); err != nil {
		return Config{}, err
	}

	return out, nil
}
