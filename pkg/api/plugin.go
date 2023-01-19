package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-plugin"
)

// HandshakeConfig is common handshake config between Botkube and its plugins.
var HandshakeConfig = plugin.HandshakeConfig{
	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "BOTKUBE",
	MagicCookieValue: "52ca7b74-28eb-4fac-ae79-31a9cbda2454",
}

// MetadataOutput contains the metadata of a given plugin.
type MetadataOutput struct {
	// Version is a version of a given plugin. It should follow the SemVer syntax.
	Version string
	// Descriptions is a description of a given plugin.
	Description string
	// JSONSchema is a JSON schema for a given plugin.
	JSONSchema JSONSchema
	// Dependencies holds the dependencies for a given platform binary.
	Dependencies map[string]Dependency
}

// JSONSchema contains the JSON schema or a remote reference where the schema can be found.
// Value and RefURL are mutually exclusive
type JSONSchema struct {
	// Value is the JSON schema string.
	Value string
	// RefURL is the remote reference of the schema.
	RefURL string
}

// Dependency holds the dependency information.
type Dependency struct {
	// URLs holds the URLs for a given dependency depending on the platform and architecture.
	URLs URLs `yaml:"urls"`
}

type URLs map[string]string

func (u URLs) For(os, arch string) (string, bool) {
	val, exists := u[fmt.Sprintf("%s/%s", os, arch)]
	return val, exists
}

// Validate validate the metadata fields and returns detected issues.
func (m MetadataOutput) Validate() error {
	var issues []string
	if m.Description == "" {
		issues = append(issues, "description field cannot be empty")
	}

	if m.Version == "" {
		issues = append(issues, "version field cannot be empty")
	}

	if m.JSONSchema.Value != "" && m.JSONSchema.RefURL != "" {
		issues = append(issues, "JSONSchema.Value and JSONSchema.RefURL are mutually exclusive. Pick one.")
	}

	if len(m.Dependencies) > 0 {
		for depKey, dep := range m.Dependencies {
			if len(dep.URLs) == 0 {
				issues = append(issues, fmt.Sprintf("dependency URLs for key %q cannot be empty", depKey))
				continue
			}

			for platformKey, platformURL := range dep.URLs {
				if platformURL != "" {
					continue
				}

				issues = append(issues, fmt.Sprintf("dependency URLs for \"%s.%s\" cannot be empty", depKey, platformKey))
			}
		}
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, ", "))
	}
	return nil
}
