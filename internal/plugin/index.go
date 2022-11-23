package plugin

import (
	"fmt"
	"strings"

	semver "github.com/hashicorp/go-version"
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
		Entries []IndexEntry
	}
	// IndexEntry defines the plugin definition.
	IndexEntry struct {
		Name        string
		Type        Type
		Description string
		Version     string
		Links       []string
	}
)

// byIndexEntryVersion implements sort.Interface based on the version field.
type byIndexEntryVersion []IndexEntry

func (a byIndexEntryVersion) Len() int      { return len(a) }
func (a byIndexEntryVersion) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIndexEntryVersion) Less(i, j int) bool {
	return semvVerGreater(a[i].Version, a[j].Version)
}

// semvVerGreater returns true if A is greater than B.
func semvVerGreater(a, b string) bool {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	verA, aErr := semver.NewVersion(a)
	verB, bErr := semver.NewVersion(b)

	return aErr == nil && bErr == nil && verA.GreaterThan(verB)
}

// BuildPluginKey returns plugin key.
func BuildPluginKey(repo, plugin string) string {
	return repo + "/" + plugin
}

// DecomposePluginKey extract details from plugin key.
func DecomposePluginKey(key string) (string, string, error) {
	repo, name, found := strings.Cut(key, "/")
	if !found {
		return "", "", fmt.Errorf("plugin %q doesn't follow required {repo_name}/{plugin_name} syntax", key)
	}
	return repo, name, nil
}
