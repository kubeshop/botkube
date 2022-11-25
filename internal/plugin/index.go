package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/multierror"
)

// Type represents the plugin type.
type Type string

const (
	// TypeSource represents the source plugin.
	TypeSource Type = "source"
	// TypeExecutor represents the executor plugin.
	TypeExecutor Type = "executor"
)

type (
	// Index defines the plugin repository index.
	Index struct {
		Entries []IndexEntry `yaml:"entries"`
	}
	// IndexEntry defines the plugin definition.
	IndexEntry struct {
		Name        string     `yaml:"name"`
		Type        Type       `yaml:"type"`
		Description string     `yaml:"description"`
		Version     string     `yaml:"version"`
		URLs        []IndexURL `yaml:"urls"`
	}

	// IndexURL holds the binary url details.
	IndexURL struct {
		URL      string           `yaml:"url"`
		Platform IndexURLPlatform `yaml:"platform"`
	}

	// IndexURLPlatform holds platform information about a given binary URL.
	IndexURLPlatform struct {
		OS   string `yaml:"os"`
		Arch string `yaml:"architecture"`
	}
)

// BuildPluginKey returns plugin key with the following format:
// <repo>/<plugin>[@<version>]
func BuildPluginKey(repo, plugin, ver string) (string, error) {
	if err := validate(repo, plugin); err != nil {
		return "", err
	}

	base := repo + "/" + plugin
	if ver != "" {
		base += "@" + ver
	}
	return base, nil
}

// DecomposePluginKey extract details from plugin key.
func DecomposePluginKey(key string) (string, string, string, error) {
	repo, name, found := strings.Cut(key, "/")
	if !found {
		return "", "", "", fmt.Errorf("plugin key %q doesn't follow required {repo_name}/{plugin_name} syntax", key)
	}

	name, ver, _ := strings.Cut(name, "@")

	if err := validate(repo, name); err != nil {
		return "", "", "", err
	}

	return repo, name, ver, nil
}

func validate(repo, plugin string) error {
	issues := multierror.New()
	if repo == "" {
		issues = multierror.Append(issues, errors.New("repository name is required"))
	}
	if plugin == "" {
		issues = multierror.Append(issues, errors.New("plugin name is required"))
	}
	return issues.ErrorOrNil()
}
