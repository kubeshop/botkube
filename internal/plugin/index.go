package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/stringx"
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
		Name             string          `yaml:"name"`
		Type             Type            `yaml:"type"`
		Description      string          `yaml:"description"`
		DocumentationURL string          `yaml:"documentationUrl,omitempty"`
		Version          string          `yaml:"version"`
		URLs             []IndexURL      `yaml:"urls"`
		JSONSchema       JSONSchema      `yaml:"jsonSchema"`
		ExternalRequest  ExternalRequest `yaml:"externalRequest,omitempty"`
		Recommended      bool            `yaml:"recommended"`
	}

	// ExternalRequest contains the external request metadata for a given plugin.
	ExternalRequest struct {
		Payload ExternalRequestPayload `yaml:"payload,omitempty"`
	}

	// ExternalRequestPayload contains the external request payload metadata for a given plugin.
	ExternalRequestPayload struct {
		// JSONSchema is a JSON schema for a given incoming webhook payload.
		JSONSchema JSONSchema `yaml:"jsonSchema"`
	}

	JSONSchema struct {
		Value  string `yaml:"value,omitempty"`
		RefURL string `yaml:"refURL,omitempty"`
	}

	// IndexURL holds the binary url details.
	IndexURL struct {
		URL          string           `yaml:"url"`
		Checksum     string           `yaml:"checksum"`
		Platform     IndexURLPlatform `yaml:"platform"`
		Dependencies Dependencies     `yaml:"dependencies,omitempty"`
	}

	// IndexURLPlatform holds platform information about a given binary URL.
	IndexURLPlatform struct {
		OS   string `yaml:"os"`
		Arch string `yaml:"architecture"`
	}

	// Dependencies holds the dependencies for a given platform binary.
	Dependencies map[string]Dependency

	// Dependency holds the dependency information.
	Dependency struct {
		URL string `yaml:"url"`
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
		for _, urlItem := range entry.URLs {
			if len(urlItem.Dependencies) == 0 {
				continue
			}
			for key, dep := range urlItem.Dependencies {
				if dep.URL != "" {
					continue
				}
				entryIssues = multierror.Append(entryIssues, fmt.Errorf("dependency URL for key %q and platform \"%s/%s\" cannot be empty", key, urlItem.Platform.OS, urlItem.Platform.Arch))
			}
		}

		if entry.Type != "" && !entry.Type.IsValid() {
			entryIssues = multierror.Append(entryIssues, fmt.Errorf("field type is not valid, allowed values are %s", allKnownTypes))
		}

		if entry.Type != "" && entry.Name != "" && entry.Version != "" {
			// check if we have a duplicate entry
			key := strings.Join([]string{string(entry.Type), entry.Name, entry.Version}, ";")
			firstEntryIdx, alreadyExist := entriesByKey[key]
			if alreadyExist {
				// duplicate, append error
				entryIssues = multierror.Append(entryIssues, fmt.Errorf("conflicts with the %s entry as both have the same type, name, and version", humanize.Ordinal(firstEntryIdx)))
				// not calling `continue` by purpose; we want to collect all the errors for the entry
			}

			entriesByKey[key] = currentIdx
		}

		err := entryIssues.ErrorOrNil()
		if err != nil {
			issues = multierror.Append(issues, fmt.Errorf("entries[%d]: %s", currentIdx, stringx.IndentAfterLine(err.Error(), 1, "\t")))
		}
	}

	return issues.ErrorOrNil()
}
