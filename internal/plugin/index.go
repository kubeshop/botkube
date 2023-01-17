package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/kubeshop/botkube/internal/stringx"
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

var allKnownTypes = []Type{TypeSource, TypeExecutor}

// IsValid checks if type is a known type.
func (t Type) IsValid() bool {
	for _, knownType := range allKnownTypes {
		if t == knownType {
			return true
		}
	}
	return false
}

// String returns a string type representation.
func (t Type) String() string {
	return string(t)
}

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
		JSONSchema  JSONSchema `yaml:"JSONSchema"`
	}

	JSONSchema struct {
		Value  string `yaml:"value"`
		RefURL string `yaml:"refURL"`
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

// Validate validates that entries define in a given index don't conflict with each other by having the same name, type and version.
func (in Index) Validate() error {
	issues := multierror.New()

	// our unique key is: type + name + version
	entriesByKey := map[string]int{}

	for currentIdx, entry := range in.Entries {
		entryIssues := multierror.New()
		if entry.Version == "" {
			entryIssues = multierror.Append(entryIssues, errors.New("field version cannot be empty"))
		}
		if entry.Name == "" {
			entryIssues = multierror.Append(entryIssues, errors.New("field name cannot be empty"))
		}
		if entry.Type == "" {
			entryIssues = multierror.Append(entryIssues, errors.New("field type cannot be empty"))
		}

		if len(entry.URLs) == 0 {
			entryIssues = multierror.Append(entryIssues, errors.New("field urls cannot be empty"))
		}

		if entry.Type != "" && !entry.Type.IsValid() {
			entryIssues = multierror.Append(entryIssues, fmt.Errorf("field type is not valid, allowed values are %s", allKnownTypes))
		}

		if entry.Type != "" && entry.Name != "" && entry.Version != "" {
			key := strings.Join([]string{string(entry.Type), entry.Name, entry.Version}, ";")
			firstEntryIdx, found := entriesByKey[key]
			if !found {
				entriesByKey[key] = currentIdx
				continue
			}

			entryIssues = multierror.Append(entryIssues, fmt.Errorf("conflicts with the %s entry as both have the same type, name, and version", humanize.Ordinal(firstEntryIdx)))
		}

		err := entryIssues.ErrorOrNil()
		if err != nil {
			issues = multierror.Append(issues, fmt.Errorf("entries[%d]: %s", currentIdx, stringx.IndentAfterLine(err.Error(), 1, "\t")))
		}
	}

	return issues.ErrorOrNil()
}
