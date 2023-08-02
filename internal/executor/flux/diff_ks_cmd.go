package flux

import (
	"strconv"
	"strings"
	"time"
)

type (
	// KustomizationCommandAliases holds different names for kustomization subcommand.
	// Unfortunately, it's a go-arg limitation that we cannot on a single entry have subcommand aliases.
	KustomizationCommandAliases struct {
		Kustomization *KustomizationDiffCommand `arg:"subcommand:kustomization"`
		Ks            *KustomizationDiffCommand `arg:"subcommand:ks"`
	}

	KustomizationDiffCommand struct {
		AppName string `arg:"positional"`
		KustomizationDiffCommandFlags
		GlobalCommandFlags
	}
)

func (k KustomizationDiffCommand) ToCmdString() string {
	return "flux diff ks " + k.AppName + k.GlobalCommandFlags.ToString() + k.KustomizationDiffCommandFlags.ToString()
}

type KustomizationDiffCommandFlags struct {
	IgnorePaths       []string `arg:"--ignore-paths,separate"`
	KustomizationFile string   `arg:"--kustomization-file"`
	Path              string   `arg:"--path"`
	ProgressBar       bool     `arg:"--progress-bar"`
	GitHubRef         string   `arg:"--github-ref"`
}

type GlobalCommandFlags struct {
	CacheDir              string        `arg:"--cache-dir"`
	DisableCompression    bool          `arg:"--disable-compression"`
	InsecureSkipTLSVerify bool          `arg:"--insecure-skip-tls-verify"`
	KubeAPIBurst          int           `arg:"--kube-api-burst"`
	KubeAPIQPS            float32       `arg:"--kube-api-qps"`
	Namespace             string        `arg:"-n,--namespace"`
	Timeout               time.Duration `arg:"--timeout"`
	Token                 string        `arg:"--token"`
	Verbose               bool          `arg:"--verbose"`
}

// Get returns HistoryCommand that were unpacked based on the alias used by user.
func (u KustomizationCommandAliases) Get() *KustomizationDiffCommand {
	if u.Kustomization != nil {
		return u.Kustomization
	}
	if u.Ks != nil {
		return u.Ks
	}

	return nil
}

func (k KustomizationDiffCommandFlags) ToString() string {
	var out strings.Builder

	if k.KustomizationFile != "" {
		out.WriteString(" --kustomization-file ")
		out.WriteString(k.KustomizationFile)
	}

	if k.Path != "" {
		out.WriteString(" --path ")
		out.WriteString(k.Path)
	}

	if len(k.IgnorePaths) != 0 {
		out.WriteString(" --ignore-paths ")
		out.WriteString(strings.Join(k.IgnorePaths, ","))
	}

	out.WriteString(" --progress-bar=false") // we don't want to have it

	return out.String()
}

func (g GlobalCommandFlags) ToString() string {
	var out strings.Builder

	if g.CacheDir != "" {
		out.WriteString(" --cache-dir ")
		out.WriteString(g.CacheDir)
	}

	if g.DisableCompression {
		out.WriteString(" --disable-compression ")
	}

	if g.InsecureSkipTLSVerify {
		out.WriteString(" --insecure-skip-tls-verify ")
	}

	if g.KubeAPIBurst != 0 {
		out.WriteString(" --kube-api-burst ")
		out.WriteString(strconv.Itoa(g.KubeAPIBurst))
	}

	if g.KubeAPIQPS != 0 {
		out.WriteString(" --kube-api-qps ")
		out.WriteString(strconv.FormatFloat(float64(g.KubeAPIQPS), 'f', -1, 32))
	}

	if g.Namespace != "" {
		out.WriteString(" -n ")
		out.WriteString(g.Namespace)
	}

	if g.Timeout != 0 {
		out.WriteString(" --timeout ")
		out.WriteString(g.Timeout.String())
	}

	if g.Token != "" {
		out.WriteString(" --token ")
		out.WriteString(g.Token)
	}

	if g.Verbose {
		out.WriteString(" --verbose ")
	}

	return out.String()
}
