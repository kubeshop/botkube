package helm

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// ListCommandAliases holds different names for list subcommand.
// Unfortunately, it's a go-arg limitation that we cannot on a single entry have subcommand aliases.
type ListCommandAliases struct {
	List *ListCommand `arg:"subcommand:list"`
	Ls   *ListCommand `arg:"subcommand:ls"`
}

// Get returns ListCommand that were unpacked based on the alias used by user.
func (u ListCommandAliases) Get() *ListCommand {
	if u.List != nil {
		return u.List
	}
	if u.Ls != nil {
		return u.Ls
	}

	return nil
}

// ListCommand holds possible uninstallation options such as positional arguments and supported flags.
// Syntax:
//
//	helm list [flags]
type ListCommand struct {
	SupportedListFlags
}

// Validate validates that all list parameters are valid.
func (ListCommand) Validate() error {
	// for now, we implemented that only to satisfy the command interface.
	return nil
}

// Help returns command help message.
func (ListCommand) Help() string {
	return heredoc.Docf(`
		Lists all of the releases for a specified namespace.

		By default, items are sorted alphabetically. Use the '-d' flag to sort by
		release date.

		If the -f flag is provided, it will be treated as a filter. Filters are
		regular expressions (Perl compatible) that are applied to the list of releases.
		Only items that match the filter will be returned.

		    helm list -f 'ara[a-z]+'

		    NAME                UPDATED                                  CHART
		    maudlin-arachnid    2020-06-18 14:17:46.125134977 +0000 UTC  alpine-0.1.0

		By default, up to 256 items may be returned. To limit this, use the '--max' flag.
		Setting '--max' to 0 will not return all results. Rather, it will return the
		server's default, which may be much higher than 256. Pairing the '--max'
		flag with the '--offset' flag allows you to page through results.

		Usage:
		  helm list [flags]

		Aliases:
		  list, ls

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedListFlags{}), 4))
}

// SupportedListFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedListFlags struct {
	All          bool   `arg:"-a,--all"`
	Namespaces   bool   `arg:"-A,--all-namespaces"`
	Date         bool   `arg:"-d,--date"`
	Deployed     bool   `arg:"--deployed"`
	Failed       bool   `arg:"--failed"`
	Max          int    `arg:"-m,--max"`
	Headers      bool   `arg:"--no-headers"`
	Offset       int    `arg:"--offset"`
	Output       string `arg:"-o,--output"`
	Pending      bool   `arg:"--pending"`
	Reverse      bool   `arg:"-r,--reverse"`
	Selector     string `arg:"-l,--selector"`
	Short        bool   `arg:"-q,--short"`
	Superseded   bool   `arg:"--superseded"`
	TimeFormat   string `arg:"--time-format"`
	Uninstalled  bool   `arg:"--uninstalled"`
	Uninstalling bool   `arg:"--uninstalling"`
	// NOTE: only the short filter flag can be used, as the --filter is already taken by the Botkube Core
	Filter string `arg:"-f"`
}
