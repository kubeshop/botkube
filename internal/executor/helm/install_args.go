package helm

import (
	"errors"
	"reflect"
	"time"
)

type InstallCmd struct {
	Name  string `arg:"positional"`
	Chart string `arg:"positional"`

	SupportedInstallFlags
	NotSupportedInstallFlags
}

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
	PostRendererArgs         []string      `arg:"--post-renderer-args postRendererArgsSlice"`
	RenderSubChartNotes      bool          `arg:"--render-subchart-notes"`
	Replace                  bool          `arg:"--replace"`
	Repo                     string        `arg:"--repo"`
	Set                      []string      `arg:"--set"`
	SetJSON                  []string      `arg:"--set-json"`
	SetString                []string      `arg:"--set-string"`
	SkipCRDs                 bool          `arg:"--skip-crds"`
	Timeout                  time.Duration `arg:"--timeout duration"`
	Username                 string        `arg:"--username"`
	Verify                   bool          `arg:"--verify"`
	Version                  string        `arg:"--version"`
}

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

var emptySupportedFlags NotSupportedInstallFlags

func (i InstallCmd) Validate() error {
	if !reflect.DeepEqual(i.NotSupportedInstallFlags, emptySupportedFlags) {
		return errors.New("user specified not supported Helm flags")
	}
	return nil
}
