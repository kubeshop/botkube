package migrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCompatible(t *testing.T) {
	testCases := []struct {
		Name                         string
		BotkubeVersionConstraintsStr string
		BotkubeVersionStr            string
		ExpectedResult               bool
		ExpectedErrMessage           string
	}{
		// TODO: The following test cases should pass, but they don't because of the bug in the `semver` library.
		//  See: https://github.com/Masterminds/semver/issues/177
		// 	The problem occurs only in the RC releases, so this isn't a huge issue for us.
		//
		//{
		//	Name:                         "RC within range",
		//	BotkubeVersionStr:            "1.8.0-rc.1",
		//	BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0-rc.1",
		//	ExpectedResult:               true,
		//	ExpectedErrMessage:           "",
		//},
		//{
		//	Name:                         "RC within range 2",
		//	BotkubeVersionStr:            "1.8.0-rc.1",
		//	BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0",
		//	ExpectedResult:               true,
		//	ExpectedErrMessage:           "",
		//},
		{
			Name:                         "Final version out of range",
			BotkubeVersionStr:            "1.8.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0-rc.1",
			ExpectedResult:               false,
			ExpectedErrMessage:           "",
		},
		{
			Name:                         "Final version within range",
			BotkubeVersionStr:            "1.8.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0",
			ExpectedResult:               true,
			ExpectedErrMessage:           "",
		},
		{
			Name:                         "Lowest version within range",
			BotkubeVersionStr:            "1.0.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0",
			ExpectedResult:               true,
			ExpectedErrMessage:           "",
		},
		{
			Name:                         "Older",
			BotkubeVersionStr:            "1.7.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0",
			ExpectedResult:               true,
			ExpectedErrMessage:           "",
		},
		{
			Name:                         "Newer",
			BotkubeVersionStr:            "1.9.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= 1.8.0",
			ExpectedResult:               false,
			ExpectedErrMessage:           "",
		},
		{
			Name:                         "Invalid constraint",
			BotkubeVersionStr:            "1.8.0",
			BotkubeVersionConstraintsStr: ">= 1.0, <= dev",
			ExpectedResult:               false,
			ExpectedErrMessage:           "unable to parse Botkube semver version constraints: improper constraint: >= 1.0, <= dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := IsCompatible(tc.BotkubeVersionConstraintsStr, tc.BotkubeVersionStr)
			if tc.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tc.ExpectedErrMessage)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}
