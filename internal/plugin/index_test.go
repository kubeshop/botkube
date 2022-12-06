package plugin

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidateIndexErrors(t *testing.T) {
	// given
	rawCfg := loadTestdataFile(t, "invalid.yaml")
	var givenIndex Index
	err := yaml.Unmarshal(rawCfg, &givenIndex)
	require.NoError(t, err)

	expErrorMsg := heredoc.Doc(`
		5 errors occurred:
			* entries[2]: 1 error occurred:
				* conflicts with the 1st entry as both have the same type, name, and version
			* entries[4]: 1 error occurred:
				* conflicts with the 3rd entry as both have the same type, name, and version
			* entries[5]: 1 error occurred:
				* field version cannot be empty
			* entries[6]: 1 error occurred:
				* field type cannot be empty
			* entries[7]: 1 error occurred:
				* field name cannot be empty`)

	// when
	err = givenIndex.Validate()

	// then
	assert.Error(t, err)
	assert.Equal(t, err.Error(), expErrorMsg)
}

func TestValidateIndexSuccess(t *testing.T) {
	// given
	rawCfg := loadTestdataFile(t, "valid.yaml")
	var givenIndex Index
	err := yaml.Unmarshal(rawCfg, &givenIndex)
	require.NoError(t, err)

	// when
	err = givenIndex.Validate()

	// then
	assert.NoError(t, err)
}
