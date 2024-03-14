package plugin

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/loggerx"
)

func TestIndexBuilder_ValidateJSONSchemas(t *testing.T) {
	// given
	index := Index{
		Entries: []IndexEntry{
			{
				Name: "value-ok",
				JSONSchema: JSONSchema{
					Value: heredoc.Doc(`
					{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"title": "botkube/helm",
						"description": "Helm",
						"type": "object",
						"properties": {
							"helmDriver": {
								"description": "Storage driver for Helm",
								"type": "string",
								"default": "secret",
								"enum": ["configmap", "secret", "memory"]
							}
						},
						"required": []
					}`),
				},
			},
			{
				Name: "ref-ok",
				JSONSchema: JSONSchema{
					RefURL: "https://json-schema.org/draft-07/schema",
				},
			},
			{
				Name: "empty",
			},
			{
				Name: "value-invalid1",
				JSONSchema: JSONSchema{
					Value: heredoc.Doc(`{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"type": "object",
						"properties": {
							"helmDriver": {
								"description": "Storage driver for Helm",
								"type": "string",
								"default": "secret",
								"enum": ["configmap", "secret", "memory"]
							},
						},
					}`),
				},
			},
			{
				Name: "value-invalid2",
				JSONSchema: JSONSchema{
					Value: "test",
				},
			},
			{
				Name: "ref-invalid",
				JSONSchema: JSONSchema{
					RefURL: "https://raw.githubusercontent.com/kubeshop/botkube/main/assets/schema.json",
				},
			},
			{
				Name: "validation-error",
				JSONSchema: JSONSchema{
					Value: heredoc.Doc(`{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"type": "object",
						"properties": {
							"helmDriver": {
								"description": "Storage driver for Helm",
								"type": "uknown",
								"enum": ["configmap", "secret", "memory"]
							}
						}
					}`),
				},
			},
		},
	}
	expectedErrMsg := heredoc.Doc(`
				5 errors occurred:
					* while validating JSON schema for "value-invalid1": invalid character '}' looking for beginning of object key string
					* while validating JSON schema for "value-invalid2": invalid character 'e' in literal true (expecting 'r')
					* while validating JSON schema for "ref-invalid": Could not read schema from HTTP, response status is 404 Not Found
					* invalid schema "validation-error": properties.helmDriver.type: Must validate at least one schema (anyOf)
					* invalid schema "validation-error": properties.helmDriver.type: properties.helmDriver.type must be one of the following: "array", "boolean", "integer", "null", "number", "object", "string"`)
	log := loggerx.NewNoop()
	bldr := NewIndexBuilder(log)

	// when
	err := bldr.validateJSONSchemas(index)

	// then
	assert.EqualError(t, err, expectedErrMsg)
}

func TestRemoveArchiveExtension(t *testing.T) {
	tests := []struct {
		name      string
		givenName string
		expName   string
	}{
		{
			name:      "Trim single extension archive",
			givenName: "test.gz",
			expName:   "test",
		},
		{
			name:      "Trim multi extension archive",
			givenName: "test.tar.gz",
			expName:   "test",
		},
		{
			name:      "Does nothing if not archive",
			givenName: "test.exe",
			expName:   "test.exe",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			gotName := trimArchiveExtension(tc.givenName)

			// then
			assert.Equal(t, tc.expName, gotName)
		})
	}
}
