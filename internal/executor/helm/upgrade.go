package helm

import (
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// UpgradeCommand holds possible upgrade options such as positional arguments and supported flags.
// Syntax:
//
//	helm upgrade [RELEASE] [CHART] [flags]
type UpgradeCommand struct {
	Name  string `arg:"positional"`
	Chart string `arg:"positional"`

	SupportedUpgradeFlags
	NotSupportedUpgradeFlags
}

// Validate validates that all list parameters are valid.
func (i UpgradeCommand) Validate() error {
	return returnErrorOfAllSetFlags(i.NotSupportedUpgradeFlags)
}

// Help returns command help message.
func (UpgradeCommand) Help() string {
	return heredoc.Docf(`
		Upgrades a release to a new version of a chart.

		The upgrade arguments must be a release and chart. The chart
		argument can be only a fully qualified URL. For chart references, the latest
		version will be specified unless the '--version' flag is set.

		To override values in a chart, use the '--set' flag and pass configuration, to force string
		values, use '--set-string'. You can also use '--set-json' to set json values
		(scalars/objects/arrays) from the command line.

		You can specify the '--set' flag multiple times. The priority will be given to the
		last (right-most) set specified. For example, if both 'bar' and 'newbar' values are
		set for a key called 'foo', the 'newbar' value would take precedence:

		    helm upgrade --set foo=bar --set foo=newbar redis https://example.com/charts/redis-1.2.3.tgz

		Usage:
		  helm upgrade [RELEASE] [CHART] [flags]
		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedUpgradeFlags{}), 4))
}

// SupportedUpgradeFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedUpgradeFlags struct {
	CreateNamespace          bool          `arg:"--create-namespace"`
	CleanupOnFail            bool          `arg:"--cleanup-on-fail"`
	DependencyUpdate         bool          `arg:"--dependency-update"`
	Description              string        `arg:"--description"`
	Devel                    bool          `arg:"--devel"`
	DisableOpenAPIValidation bool          `arg:"--disable-openapi-validation"`
	Force                    bool          `arg:"--force"`
	DryRun                   bool          `arg:"--dry-run"`
	HistoryMax               int           `arg:"--history-max"`
	InsecureSkipTLSVerify    bool          `arg:"--insecure-skip-tls-verify"`
	Install                  bool          `arg:"--install,-i"`
	NoHooks                  bool          `arg:"--no-hooks"`
	PassCredentials          bool          `arg:"--pass-credentials"`
	Password                 string        `arg:"--password"`
	PostRenderer             string        `arg:"--post-renderer"`
	PostRendererArgs         []string      `arg:"--post-renderer-args"`
	RenderSubChartNotes      bool          `arg:"--render-subchart-notes"`
	Repo                     string        `arg:"--repo"`
	Set                      []string      `arg:"--set"`
	SetJSON                  []string      `arg:"--set-json"`
	SetString                []string      `arg:"--set-string"`
	SkipCRDs                 bool          `arg:"--skip-crds"`
	Timeout                  time.Duration `arg:"--timeout"`
	Username                 string        `arg:"--username"`
	Verify                   bool          `arg:"--verify"`
	ResetValues              bool          `arg:"--reset-values"`
	ReuseValues              bool          `arg:"--reuse-values"`
	Version                  string        `arg:"--version"`
	Output                   string        `arg:"-o,--output"`
}

// NotSupportedUpgradeFlags represents flags supported by Helm CLI but not by Helm Plugin.
type NotSupportedUpgradeFlags struct {
	Atomic      bool     `arg:"--atomic"`
	CaFile      string   `arg:"--ca-file"`
	CertFile    string   `arg:"--cert-file"`
	KeyFile     string   `arg:"--key-file"`
	Keyring     string   `arg:"--keyring"`
	SetFile     []string `arg:"--set-file"`
	Values      []string `arg:"-f,--values"`
	Wait        bool     `arg:"--wait"`
	WaitForJobs bool     `arg:"--wait-for-jobs"`
}
