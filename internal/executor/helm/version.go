package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// VersionCommand holds possible version options such as positional arguments and supported flags.
// Syntax:
//
//	helm version [flags]
type VersionCommand struct {
	noopValidator

	SupportedVersionFlags
}

// Help returns command help message.
func (VersionCommand) Help() string {
	return heredoc.Docf(`
		Shows the version of the Helm CLI used by this Botkube plugin.

		The output will look something like this:

		version.BuildInfo{Version:"v3.2.1", GitCommit:"fe51cd1e31e6a202cba7dead9552a6d418ded79a", GitTreeState:"clean", GoVersion:"go1.13.10"}

		- Version is the semantic version of the release.
		- GitCommit is the SHA for the commit that this version was built from.
		- GitTreeState is "clean" if there are no local code changes when this binary was
		  built, and "dirty" if the binary was built from locally modified code.
		- GoVersion is the version of Go that was used to compile Helm.

		Usage:
		  helm version [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedVersionFlags{}), 4))
}

// SupportedVersionFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedVersionFlags struct {
	Short    bool   `arg:"--short"`
	Template string `arg:"--template"`
}
