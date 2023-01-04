package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// GetCommand holds possible get options such as positional arguments and supported flags.
// Syntax:
//
//	helm get [command]
type GetCommand struct {
	All      *GetAllCommand      `arg:"subcommand:all"`
	Hooks    *GetHooksCommand    `arg:"subcommand:hooks"`
	Manifest *GetManifestCommand `arg:"subcommand:manifest"`
	Notes    *GetNotesCommand    `arg:"subcommand:notes"`
	Values   *GetValuesCommand   `arg:"subcommand:values"`

	SupportedGetFlags
}

// SupportedGetFlags represents flags that are supported both by Helm CLI and Helm Plugin.
type SupportedGetFlags struct {
	Revision int `arg:"--revision"`
}

// Help returns command help message.
func (GetCommand) Help() string {
	return heredoc.Doc(`
		This command consists of multiple subcommands which can be used to
		get extended information about the release, including:

		- The values used to generate the release
		- The generated manifest file
		- The notes provided by the chart of the release
		- The hooks associated with the release

		Usage:
		  helm get [command]

		Available Commands:
		  all         # Shows all information for a named release
		  hooks       # Shows all hooks for a named release
		  manifest    # Shows the manifest for a named release
		  notes       # Shows the notes for a named release
		  values      # Shows the values file for a named release

		Use "helm get [command] --help" for more information about the command.
	`)
}

// GetAllCommand holds possible get options such as positional arguments and supported flags.
type GetAllCommand struct {
	noopValidator
	Name string `arg:"positional"`

	SupportedGetAllFlags
}

// SupportedGetAllFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedGetAllFlags struct {
	Template string `arg:"--template"`
}

// Help returns command help message.
func (GetAllCommand) Help() string {
	return heredoc.Docf(`
		Shows a human readable collection of information about the
		notes, hooks, supplied values, and generated manifest file of the given release.

		Usage:
		  helm get all RELEASE_NAME [flags]
		Flags:
		%s
		%s
	`,
		indent.String(renderSupportedFlags(SupportedGetFlags{}), 4),    // root flags
		indent.String(renderSupportedFlags(SupportedGetAllFlags{}), 4), // specific values flags
	)
}

// GetHooksCommand holds possible get options such as positional arguments and supported flags.
type GetHooksCommand struct {
	noopValidator
	Name string `arg:"positional"`
}

// Help returns command help message.
func (GetHooksCommand) Help() string {
	return heredoc.Docf(`
		Shows hooks for a given release.

		Hooks are formatted in YAML and separated by the YAML '---\n' separator.

		Usage:
		  helm get hooks RELEASE_NAME [flags]
		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedGetFlags{}), 4))
}

// GetManifestCommand holds possible get options such as positional arguments and supported flags.
type GetManifestCommand struct {
	noopValidator
	Name string `arg:"positional"`
}

// Help returns command help message.
func (GetManifestCommand) Help() string {
	return heredoc.Docf(`
		Shows the generated manifest for a given release.

		A manifest is a YAML-encoded representation of the Kubernetes resources that
		were generated from this release's chart(s). If a chart is dependent on other
		charts, those resources will also be included in the manifest.

		Usage:
		  helm get manifest RELEASE_NAME [flags]
		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedGetFlags{}), 4))
}

// GetNotesCommand holds possible get options such as positional arguments and supported flags.
type GetNotesCommand struct {
	noopValidator
	Name string `arg:"positional"`
}

// Help returns command help message.
func (GetNotesCommand) Help() string {
	return heredoc.Docf(`
		Shows notes provided by the chart of a named release.

		Usage:
		  helm get notes RELEASE_NAME [flags]
		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedGetFlags{}), 4))
}

// GetValuesCommand holds possible get options such as positional arguments and supported flags.
type GetValuesCommand struct {
	noopValidator
	Name string `arg:"positional"`

	SupportedGetValuesFlags
}

// SupportedGetValuesFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedGetValuesFlags struct {
	All    bool   `arg:"-a,--all"`
	Output string `arg:"-o,--output"`
}

// Help returns command help message.
func (GetValuesCommand) Help() string {
	return heredoc.Docf(`
		Shows a values file for a given release.

		Usage:
		  helm get values RELEASE_NAME [flags]
		Flags:
		%s
		%s
	`,
		indent.String(renderSupportedFlags(SupportedGetFlags{}), 4),       // root flags
		indent.String(renderSupportedFlags(SupportedGetValuesFlags{}), 4), // specific values flags
	)
}
