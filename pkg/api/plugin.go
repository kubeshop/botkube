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
	// URL to plugin documentation.
	DocumentationURL string
	// JSONSchema is a JSON schema for a given plugin configuration.
	JSONSchema JSONSchema

	// ExternalRequest holds the metadata for external requests.
	ExternalRequest ExternalRequestMetadata

	// Dependencies holds the dependencies for a given platform binary.
	Dependencies map[string]Dependency
	// Recommended says if plugin is recommended
	Recommended bool
}

// ExternalRequestMetadata contains the metadata for external requests.
type ExternalRequestMetadata struct {
	// Payload contains the external requests payload information.
	Payload ExternalRequestPayload
}

// ExternalRequestPayload contains the incoming webhook payload information.
type ExternalRequestPayload struct {
	// JSONSchema is a JSON schema for a given incoming webhook payload.
	JSONSchema JSONSchema
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

// URLs is a map of URLs for different platform and architecture.
// The key format is "{os}/{arch}".
type URLs map[string]string

// For returns the URL for a given platform and architecture.
func (u URLs) For(os, arch string) (string, bool) {
	key := fmt.Sprintf("%s/%s", os, arch)
	val, exists := u[key]
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

// PluginDependencyURLsGetter is an interface for getting plugin dependency URLs.
type PluginDependencyURLsGetter interface {
	GetUrls() map[string]string
}

// ConvertDependenciesToAPI converts source/executor plugin dependencies to API dependencies.
func ConvertDependenciesToAPI[T PluginDependencyURLsGetter](resp map[string]T) map[string]Dependency {
	dependencies := make(map[string]Dependency, len(resp))
	for depName, depDetails := range resp {
		dependencies[depName] = Dependency{
			URLs: depDetails.GetUrls(),
		}
	}

	return dependencies
}

// PluginDependencyURLsSetter is an interface for setting plugin dependency URLs.
type PluginDependencyURLsSetter[T any] interface {
	SetUrls(in map[string]string)
	*T // This is needed to ensure we can create an instance of the concrete type as a part of the ConvertDependenciesFromAPI function.
}

// ConvertDependenciesFromAPI converts API dependencies to source/executor plugin dependencies.
func ConvertDependenciesFromAPI[T PluginDependencyURLsSetter[P], P any](in map[string]Dependency) map[string]T {
	dependencies := make(map[string]T, len(in))
	for depName, depDetails := range in {
		// See https://stackoverflow.com/questions/69573113/how-can-i-instantiate-a-non-nil-pointer-of-type-argument-with-generic-go
		var underlying P
		dep := T(&underlying)
		dep.SetUrls(depDetails.URLs)
		dependencies[depName] = dep
	}

	return dependencies
}
