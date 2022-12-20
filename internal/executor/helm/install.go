package helm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/muesli/reflow/indent"
)

const tagArgName = "arg"

// InstallCommand holds possible installation options such as positional arguments and supported flags
// Syntax:
//
//	helm install [NAME] [CHART] [flags]
type InstallCommand struct {
	Name  string `arg:"positional"`
	Chart string `arg:"positional"`

	SupportedInstallFlags
	NotSupportedInstallFlags
}

// SupportedInstallFlags represent flags that are supported both by Helm CLI and Helm Plugin.
type SupportedInstallFlags struct {
	CreateNamespace          bool          `arg:"--create-namespace"`
	GenerateName             bool          `arg:"--generate-name,-g"`
	DependencyUpdate         bool          `arg:"--dependency-update"`
	DescriptionD             string        `arg:"--description"`
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
}

// Validate validates that all installation parameters are valid.
func (i InstallCommand) Validate() error {
	if strings.HasPrefix(i.Chart, "oci://") {
		return errors.New("Installing Helm chart from OCI registry is not supported.")
	}
	if err := i.NotSupportedInstallFlags.Validate(); err != nil {
		return err
	}

	return nil
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
	Output      string   `arg:"-o,--output"`
}

// Validate returns an error if some unsupported flags were specified.
func (f NotSupportedInstallFlags) Validate() error {
	var setFlags []string
	vv := reflect.ValueOf(f)
	fields := reflect.VisibleFields(reflect.TypeOf(f))

	for _, field := range fields {
		flagName, _ := field.Tag.Lookup(tagArgName)
		if vv.FieldByIndex(field.Index).IsZero() {
			continue
		}

		setFlags = append(setFlags, flagName)
	}

	if len(setFlags) > 0 {
		return newUnsupportedFlagsError(setFlags)
	}
	return nil
}

func newUnsupportedFlagsError(flags []string) error {
	if len(flags) == 1 {
		return fmt.Errorf("The %q flag is not supported by the Botkube Helm plugin. Please remove it.", flags[0])
	}

	points := make([]string, len(flags))
	for i, err := range flags {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Errorf(
		"Those flags are not supported by the Botkube Helm Plugin:\n\t%s\nPlease remove them.",
		strings.Join(points, "\n\t"))
}

func renderSupportedFlags() string {
	var flags []string
	fields := reflect.VisibleFields(reflect.TypeOf(SupportedInstallFlags{}))
	for _, field := range fields {
		flagName, _ := field.Tag.Lookup(tagArgName)
		flags = append(flags, flagName)
	}

	return strings.Join(flags, "\n")
}

func helpInstall() string {
	return heredoc.Docf(`
		This command installs a chart archive.

		There are two different ways you to install a Helm chart:
		1. By absolute URL: helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
		2. By chart reference and repo url: helm install --repo https://example.com/charts/ mynginx nginx

		Usage:
		    helm install [NAME] [CHART] [flags]

		Flags:
		%s
	`, indent.String(renderSupportedFlags(), 4))
}
