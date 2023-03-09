package plugin

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/internal/loggerx"
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
						"$schema": "http://json-schema.org/draft-04/schema#",
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
						"$schema": "http://json-schema.org/draft-04/schema#",
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
					RefURL: "http://example.com/invalid-schema/",
				},
			},
		},
	}
	expectedErrMsg := heredoc.Doc(`
				3 errors occurred:
					* while validating JSON schema for value-invalid1: invalid character '}' looking for beginning of object key string
					* while validating JSON schema for value-invalid2: invalid character 'e' in literal true (expecting 'r')
					* while validating JSON schema for ref-invalid: Could not read schema from HTTP, response status is 404 Not Found`)
	log := loggerx.NewNoop()
	bldr := NewIndexBuilder(log)

	// when
	err := bldr.validateJSONSchemas(index)

	// then
	assert.EqualError(t, err, expectedErrMsg)
}
