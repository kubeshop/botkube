package config

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/kubeshop/botkube/pkg/multierror"
)

// BuildPluginKey returns plugin key with the following format:
// <repo>/<plugin>[@<version>]
func BuildPluginKey(repo, plugin, ver string) (string, error) {
	if err := validatePluginProperties(repo, plugin); err != nil {
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

	if err := validatePluginProperties(repo, name); err != nil {
		return "", "", "", err
	}

	return repo, name, ver, nil
}

func validatePluginProperties(repo, plugin string) error {
	issues := multierror.New()
	if repo == "" {
		issues = multierror.Append(issues, errors.New("repository name is required"))
	}
	if plugin == "" {
		issues = multierror.Append(issues, errors.New("plugin name is required"))
	}
	return issues.ErrorOrNil()
}

type validatePluginEntry struct {
	Repo    string
	Version string
}

// validateBindPlugins validates that only unique plugins are on the bind list.
//
// NOTE: We use a strict matching. We don't support `botkube/kubectl` and botkube/kubectl@v1.1.0 even thought it may resolve to the same version
// because if version is not specified then we use the latest one found in a given repository which may be v1.1.0.
func validateBindPlugins(sl validator.StructLevel, enabledPluginsViaBindings []string) {
	indexedByName := map[string]validatePluginEntry{}

	for _, key := range enabledPluginsViaBindings {
		repo, name, ver, err := DecomposePluginKey(key)
		if err != nil {
			// TODO: problems with keys are reported already via 'executor' configuration validator
			continue
		}

		newEntry := validatePluginEntry{
			Repo:    repo,
			Version: ver,
		}

		alreadyIndexed, found := indexedByName[name]
		if !found {
			indexedByName[name] = newEntry
			continue
		}

		if alreadyIndexed.Repo != newEntry.Repo {
			msg := fmt.Sprintf("conflicts with already bind %q plugin from %q repository. Bind it to a different channel, or change it to the one from the %q repository, or remove it.", name, alreadyIndexed.Repo, alreadyIndexed.Repo)
			sl.ReportError(key, key, key, conflictingPluginRepoTag, msg)
			continue
		}
		if alreadyIndexed.Version != newEntry.Version {
			verInfo := "the latest version" // if version not specified, we search for the latest plugin version in a given repository.
			if alreadyIndexed.Version != "" {
				verInfo = fmt.Sprintf("the %q version", alreadyIndexed.Version)
			}
			msg := fmt.Sprintf("conflicts with already bind %q plugin in %s. Bind it to a different channel, or change it to %s, or remove it.", name, verInfo, verInfo)

			sl.ReportError(key, key, key, conflictingPluginVersionTag, msg)
		}
	}
}

func validatePlugins(sl validator.StructLevel, pluginConfigs PluginsExecutors) {
	var enabledPluginsViaBindings []string
	for pluginKey, plugin := range pluginConfigs {
		if !plugin.Enabled {
			continue
		}

		enabledPluginsViaBindings = append(enabledPluginsViaBindings, pluginKey)
	}
	sort.Strings(enabledPluginsViaBindings)

	indexedByName := map[string]validatePluginEntry{}

	for _, key := range enabledPluginsViaBindings {
		repo, name, ver, err := DecomposePluginKey(key)
		if err != nil {
			// TODO: problems with keys are reported already via 'executor' configuration validator
			continue
		}

		newEntry := validatePluginEntry{
			Repo:    repo,
			Version: ver,
		}

		alreadyIndexed, found := indexedByName[name]
		if !found {
			indexedByName[name] = newEntry
			continue
		}

		if alreadyIndexed.Repo != newEntry.Repo {
			msg := fmt.Sprintf("conflicts with already defined %q plugin from %q repository. Extract it to a dedicated configuration group or remove it from this one.", name, alreadyIndexed.Repo)
			sl.ReportError(key, key, key, conflictingPluginRepoTag, msg)
			continue
		}
		if alreadyIndexed.Version != newEntry.Version {
			verInfo := "the latest version" // if version not specified, we search for the latest plugin version in a given repository.
			if alreadyIndexed.Version != "" {
				verInfo = fmt.Sprintf("the %q version", alreadyIndexed.Version)
			}
			msg := fmt.Sprintf("conflicts with already defined %q plugin in %s. Extract it to a dedicated configuration group or remove it from this one.", name, verInfo)

			sl.ReportError(key, key, key, conflictingPluginVersionTag, msg)
		}
	}
}
