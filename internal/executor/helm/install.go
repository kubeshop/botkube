package helm

import (
	"errors"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

// InstallCommand holds possible installation options such as positional arguments and supported flags.
// Syntax:
//
//	helm install [NAME] [CHART] [flags]
type InstallCommand struct {
	Name  string `arg:"positional"`
	Chart string `arg:"positional"`

	SupportedInstallFlags
	NotSupportedInstallFlags
}

// Validate validates that all installation parameters are valid.
func (i InstallCommand) Validate() error {
	if strings.HasPrefix(i.Chart, "oci://") {
		return errors.New("Installing Helm chart from OCI registry is not supported.")
	}
	if err := returnErrorOfAllSetFlags(i.NotSupportedInstallFlags); err != nil {
		return err
	}

	return nil
}

// Help returns command help message.
func (InstallCommand) Help() string {
	return heredoc.Docf(`
		Installs a chart archive.

		There are two different ways you to install a Helm chart:
		1. By absolute URL: helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
		2. By chart reference and repo url: helm install --repo https://example.com/charts/ mynginx nginx

		Usage:
		    helm install [NAME] [CHART] [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(SupportedInstallFlags{}), 4))
}

// SupportedInstallFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedInstallFlags struct {
	CreateNamespace          bool          `arg:"--create-namespace"`
	GenerateName             bool          `arg:"--generate-name,-g"`
	DependencyUpdate         bool          `arg:"--dependency-update"`
	Description              string        `arg:"--description"`
	Devel                    bool          `arg:"--devel"`
	DisableOpenAPIValidation bool          `arg:"--disable-openapi-validation"`
	DryRun                   bool          `arg:"--dry-run"`
	InsecureSkipTLSVerify    bool          `arg:"--insecure-skip-tls-verify"`
	NameTemplate             string        `arg:"--name-template"`
	NoHooks                  bool          `arg:"--no-hooks"`
	PassCredentials          bool          `arg:"--pass-credentials"`
	Password                 string        `arg:"--password"`
	PostRenderer             string        `arg:"--post-renderer"`
	PostRendererArgs         []string      `arg:"--post-renderer-args"`
	RenderSubChartNotes      bool          `arg:"--render-subchart-notes"`
	Replace                  bool          `arg:"--replace"`
	Repo                     string        `arg:"--repo"`
	Set                      []string      `arg:"--set"`
	SetJSON                  []string      `arg:"--set-json"`
	SetString                []string      `arg:"--set-string"`
	SkipCRDs                 bool          `arg:"--skip-crds"`
	Timeout                  time.Duration `arg:"--timeout"`
	Username                 string        `arg:"--username"`
	Verify                   bool          `arg:"--verify"`
	Version                  string        `arg:"--version"`
	Output                   string        `arg:"-o,--output"`
}

// NotSupportedInstallFlags represents flags supported by Helm CLI but not by Helm Plugin.
type NotSupportedInstallFlags struct {
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
