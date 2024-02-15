package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFirstFileInDirectory(t *testing.T) {
	tests := []struct {
		name             string
		dir              string
		expectedFileName string
		expectedError    error
	}{
		{
			name:             "NoFilesInDirectory",
			dir:              "./testdata/TestGetFirstFileInDirectory",
			expectedFileName: "",
			expectedError:    nil,
		},
		{
			name:             "HiddenAndExeFiles",
			dir:              "./testdata/TestGetFirstFileInDirectory/hidden_and_exe_files",
			expectedFileName: "test.exe",
			expectedError:    nil,
		},
		{
			name:             "ValidFileFound",
			dir:              "./testdata/TestGetFirstFileInDirectory/valid_file",
			expectedFileName: "example_bin",
			expectedError:    nil,
		},
		{
			name:             "BashScript",
			dir:              "./testdata/TestGetFirstFileInDirectory/valid_script",
			expectedFileName: "script.sh",
			expectedError:    nil,
		},
		{
			name:             "NoValidFile",
			dir:              "./testdata/TestGetFirstFileInDirectory/no_valid_file",
			expectedFileName: "",
			expectedError:    nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			gotFileName, err := getFirstFileInDirectory(tc.dir)

			// then
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedFileName, gotFileName)
		})
	}
}
