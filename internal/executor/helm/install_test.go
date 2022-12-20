package helm

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
)

func TestValidateNotSupportedFlags(t *testing.T) {
	tests := []struct {
		name   string
		flags  NotSupportedInstallFlags
		errMsg string
	}{
		{
			name: "all flags set",
			flags: NotSupportedInstallFlags{
				Atomic:      true,
				CaFile:      "./some-file",
				CertFile:    "./some-file",
				KeyFile:     "./somefile",
				Keyring:     "keyring",
				SetFile:     []string{"file"},
				Values:      []string{"./values.yaml"},
				Wait:        true,
				WaitForJobs: true,
				Output:      "yaml",
			},
			errMsg: heredoc.Doc(`
					Those flags are not supported by the Botkube Helm Plugin:
						* --atomic
						* --ca-file
						* --cert-file
						* --key-file
						* --keyring
						* --set-file
						* -f,--values
						* --wait
						* --wait-for-jobs
						* -o,--output
					Please remove them.`),
		},
		{
			name: "3 flags set",
			flags: NotSupportedInstallFlags{
				Atomic:  true,
				CaFile:  "./some-file",
				KeyFile: "./somefile",
				SetFile: []string{"file"},
			},
			errMsg: heredoc.Doc(`
					Those flags are not supported by the Botkube Helm Plugin:
						* --atomic
						* --ca-file
						* --key-file
						* --set-file
					Please remove them.`),
		},
		{
			name: "1 flags set",
			flags: NotSupportedInstallFlags{
				Atomic: true,
			},
			errMsg: `The "--atomic" flag is not supported by the Botkube Helm plugin. Please remove it.`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			givenCommand := InstallCommand{
				NotSupportedInstallFlags: tc.flags,
			}

			// when
			gotErr := givenCommand.Validate()

			// then
			assert.EqualError(t, gotErr, tc.errMsg)
		})
	}
}
