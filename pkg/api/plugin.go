package api

import (
	"errors"
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
}

// JSONSchema contains the JSON schema or a remote reference where the schema can be found.
// Value and RefURL are mutually exclusive
type JSONSchema struct {
	// Value is the JSON schema string.
	Value string
	// RefURL is the remote reference of the schema.
	RefURL string
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

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, ", "))
	}
	return nil
}
