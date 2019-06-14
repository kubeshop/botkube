package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventTypeString(t *testing.T) {
	tests := map[string]struct {
		givenEvent     EventType
		expextedString string
	}{
		"sample event": {EventType("sample event"), "sample event"},
		"empty event":  {EventType(""), ""},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expextedString, test.givenEvent.String())
		})
	}
}

func TestNew(t *testing.T) {
	tests := map[string]struct {
		configPath   string
		fileContent  string
		createFile   bool
		expectConfig bool
		expectError  bool
	}{
		"config file not present":       {"/tmp", "", false, true, true},
		"empty config file present":     {"/tmp", "", true, true, false},
		"non empty config file present": {"/tmp", "{}", true, true, false},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			if test.createFile {
				err := ioutil.WriteFile(filepath.Clean(filepath.Join(test.configPath,
					ConfigFileName)), []byte(test.fileContent), 0755)
				if err != nil {
					t.Errorf("Unable to write file: %v", err)
				}
			}
			os.Setenv("CONFIG_PATH", test.configPath)
			cfg, err := New()
			if test.expectConfig != (cfg != nil) {
				t.Errorf("Expect config %t but got %t", test.expectConfig, (cfg != nil))
			}
			if test.expectError != (err != nil) {
				t.Errorf("Expect error %t but got %t", test.expectError, (err != nil))
			}
			os.Unsetenv("CONFIG_PATH")
			if test.createFile {
				err := os.Remove(filepath.Clean(filepath.Join(test.configPath, ConfigFileName)))
				if err != nil {
					t.Errorf("Unable to remove file: %v", err)
				}

			}
		})
	}
}
