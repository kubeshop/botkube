package install

import (
	"context"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/heredoc"
	"github.com/kubeshop/botkube/internal/cli/install/helm"
	"github.com/kubeshop/botkube/internal/cli/printer"
)

// Install installs Botkube Helm chart into cluster.
func Install(ctx context.Context, w io.Writer, k8sCfg *rest.Config, opts Config) (err error) {
	status := printer.NewStatus(w, "Installing Botkube on cluster...")
	defer func() {
		status.End(err == nil)
	}()

	switch opts.HelmParams.RepoLocation {
	case StableVersionTag:
		opts.HelmParams.RepoLocation = HelmRepoStable
		if opts.HelmParams.Version == LatestVersionTag {
			ver, err := helm.GetLatestVersion(opts.HelmParams.RepoLocation, opts.HelmParams.ChartName)
			if err != nil {
				return err
			}
			opts.HelmParams.Version = ver
		}
	case LocalVersionTag:
		opts.HelmParams.RepoLocation = LocalChartsPath
	}

	if cli.VerboseMode.IsEnabled() {
		status.InfoWithBody("Installation details:", installDetails(opts))
	}

	helmInstaller, err := helm.NewHelm(k8sCfg, opts.HelmParams.Namespace)
	if err != nil {
		return err
	}

	status.Step("Creating namespace %s", opts.HelmParams.Namespace)
	err = ensureNamespaceCreated(ctx, k8sCfg, opts.HelmParams.Namespace)
	if err != nil {
		return err
	}

	//log.SetOutput(io.Discard)
	status.Step("Installing %s Helm chart", opts.HelmParams.ChartName)
	rel, err := helmInstaller.Install(ctx, opts.HelmParams)
	status.End(err == nil)
	if err != nil {
		return err
	}

	if cli.VerboseMode.IsEnabled() {
		desc := helm.GetStringStatusFromRelease(rel)
		fmt.Fprintln(w, desc)
	}

	welcomeMessage(w)

	return nil
}

func welcomeMessage(w io.Writer) {
	msg := heredoc.Docf(`
		Botkube installed successfully!
		
		To read more how to use CLI, check out the documentation on https://docs.botkube.io/docs/cli/getting-started#first-use.`)
	fmt.Fprintln(w, msg)
}

func installDetails(opts Config) string {
	out := &strings.Builder{}
	fmt.Fprintf(out, "\tVersion: %s\n", opts.HelmParams.Version)
	fmt.Fprintf(out, "\tHelm repository: %s\n", opts.HelmParams.RepoLocation)

	return out.String()
}

// ensureNamespaceCreated creates a k8s namespaces. If it already exists it does nothing.
func ensureNamespaceCreated(ctx context.Context, config *rest.Config, namespace string) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	nsName := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(ctx, nsName, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
