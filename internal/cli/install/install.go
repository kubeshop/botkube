package install

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/morikuni/aec"
	"github.com/muesli/reflow/indent"
	"go.szostok.io/version/style"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/install/helm"
	"github.com/kubeshop/botkube/internal/cli/install/logs"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/kubex"
)

const messageInitialBufferSize = 100

// Install installs Botkube Helm chart into cluster.
func Install(ctx context.Context, w io.Writer, k8sCfg *kubex.ConfigWithMeta, opts Config) (err error) {
	ctxWithTimeout, cancel := context.WithCancel(ctx)
	if opts.Timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctxWithTimeout, opts.Timeout)
	}
	defer cancel()

	status := printer.NewStatus(w, "Installing Botkube on cluster...")
	defer func() {
		status.End(err == nil)
		fmt.Println(aec.Show)
	}()

	switch opts.HelmParams.RepoLocation {
	case StableVersionTag:
		status.Debugf("Resolved %s tag into %s...", StableVersionTag, HelmRepoStable)
		opts.HelmParams.RepoLocation = HelmRepoStable
	case LocalVersionTag:
		status.Debugf("Resolved %s tag into %s...", LocalVersionTag, LocalChartsPath)
		opts.HelmParams.RepoLocation = LocalChartsPath
		opts.HelmParams.Version = ""
	}

	if opts.HelmParams.Version == LatestVersionTag {
		ver, err := helm.GetLatestVersion(opts.HelmParams.RepoLocation, opts.HelmParams.ChartName)
		if err != nil {
			return err
		}
		status.Debugf("Resolved %s tag into %s...", LatestVersionTag, ver)
		opts.HelmParams.Version = ver
	}

	if err = printInstallationDetails(k8sCfg, opts, status); err != nil {
		return err
	}

	helmInstaller, err := helm.NewHelm(k8sCfg.K8s, opts.HelmParams.Namespace)
	if err != nil {
		return err
	}

	status.Step("Creating namespace %s", opts.HelmParams.Namespace)
	clientset, err := kubernetes.NewForConfig(k8sCfg.K8s)
	if err != nil {
		return err
	}

	err = ensureNamespaceCreated(ctxWithTimeout, clientset, opts.HelmParams.Namespace)
	status.End(err == nil)
	if err != nil {
		return err
	}

	parallel, _ := errgroup.WithContext(ctxWithTimeout)

	podScheduledIndicator := make(chan string)
	podWaitResult := make(chan error, 1)
	parallel.Go(func() error {
		err := kubex.WaitForPod(ctxWithTimeout, clientset, opts.HelmParams.Namespace, opts.HelmParams.ReleaseName, kubex.PodReady(podScheduledIndicator, time.Now()))
		podWaitResult <- err
		return nil
	})

	rel, err := helmInstaller.Install(ctxWithTimeout, status, opts.HelmParams)
	if err != nil {
		return err
	}

	if !opts.Watch {
		status.Infof("Watching Botkube installation is disabled")
		if err := helm.PrintReleaseStatus("Release details:", status, rel); err != nil {
			return err
		}

		return printSuccessInstallMessage(opts.HelmParams.Version, w)
	}

	status.Step("Waiting until Botkube Pod is running")
	var podName string
	select {
	case podName = <-podScheduledIndicator:
		status.End(true)
	case <-time.After(opts.Timeout):
		return fmt.Errorf("Timed out waiting for Pod")
	}

	messages := make(chan []byte, messageInitialBufferSize)
	streamLogCtx, cancelStreamLogs := context.WithCancel(context.Background())
	defer cancelStreamLogs()
	parallel.Go(func() error {
		defer close(messages)
		return logs.StartsLogsStreaming(streamLogCtx, clientset, opts.HelmParams.Namespace, podName, messages)
	})

	logsPrinter := logs.NewPrinter(
		podName,
	)

	parallel.Go(func() error {
		logsPrinter.Start(ctxWithTimeout, status)
		return nil
	})
	parallel.Go(func() error {
		for {
			select {
			case <-ctxWithTimeout.Done(): // it's canceled on OS signals or if function passed to 'Go' method returns a non-nil error
				return ctxWithTimeout.Err()
			case err := <-podWaitResult:
				time.Sleep(time.Second)
				cancelStreamLogs()
				return err
			}
		}
	})

	parallel.Go(func() error {
		for {
			select {
			case <-ctxWithTimeout.Done(): // it's canceled on OS signals or if function passed to 'Go' method returns a non-nil error
				return ctxWithTimeout.Err()
			case entry, ok := <-messages:
				if !ok {
					logsPrinter.Stop()
					return nil
				}
				logsPrinter.AppendLogEntry(string(entry))
			}
		}
	})

	err = parallel.Wait()
	if err != nil {
		printErr := printFailedInstallMessage(opts.HelmParams.Version, opts.HelmParams.Namespace, podName, w)
		if printErr != nil {
			return fmt.Errorf("%s: %v", printErr, err)
		}
		return err
	}

	if err := helm.PrintReleaseStatus("Release details:", status, rel); err != nil {
		return err
	}

	return printSuccessInstallMessage(opts.HelmParams.Version, w)
}

var successInstallGoTpl = `

  │ Botkube {{ .Version | Bold }} installed successfully! 🚀
  │ To read more how to use CLI, check out the documentation on {{ .DocsURL  | Underline | Blue }}
`

func printSuccessInstallMessage(version string, w io.Writer) error {
	renderer := style.NewGoTemplateRender(style.DefaultConfig(successInstallGoTpl))

	props := map[string]string{
		"DocsURL": "https://docs.botkube.io/cli/getting-started/#first-use",
		"Version": version,
	}

	out, err := renderer.Render(props, cli.IsSmartTerminal(w))
	if err != nil {
		return fmt.Errorf("while rendering message: %v", err)
	}

	_, err = fmt.Fprint(w, out)
	if err != nil {
		return err
	}

	return nil
}

var failedInstallGoTpl = `
  │ {{ printf "Botkube %s installation failed 😿" .Version | Bold | Red }}
  │ To get all Botkube logs, run:
  │
  │   kubectl logs -n {{ .Namespace }} pod/{{ .PodName }}

  │ To receive assistance, please join our Slack community at {{ .SlackURL  | Underline | Blue }}. 
  │ We'll be glad to help you get Botkube up and running!
`

func printFailedInstallMessage(version string, namespace string, name string, w io.Writer) error {
	renderer := style.NewGoTemplateRender(style.DefaultConfig(failedInstallGoTpl))

	props := map[string]string{
		"SlackURL":  "https://join.botkube.io",
		"Version":   version,
		"Namespace": namespace,
		"PodName":   name,
	}

	out, err := renderer.Render(props, cli.IsSmartTerminal(w))
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, out)
	if err != nil {
		return fmt.Errorf("while printing message: %v", err)
	}

	return nil
}

var infoFieldsGoTpl = `{{ AdjustKeyWidth . }}
  {{- range $item := (. | Extra) }}
  {{ $item.Key | Key   }}    {{ $item.Value | Val }}
  {{- end}}

`

type Custom struct {
	// Fields are printed in the same order as defined in struct.
	Version  string `pretty:"Version"`
	HelmRepo string `pretty:"Helm repository"`
	K8sCtx   string `pretty:"Kubernetes Context"`
}

func printInstallationDetails(cfg *kubex.ConfigWithMeta, opts Config, status *printer.StatusPrinter) error {
	renderer := style.NewGoTemplateRender(style.DefaultConfig(infoFieldsGoTpl))

	out, err := renderer.Render(Custom{
		Version:  opts.HelmParams.Version,
		HelmRepo: opts.HelmParams.RepoLocation,
		K8sCtx:   cfg.CurrentContext,
	}, cli.IsSmartTerminal(status.Writer()))
	if err != nil {
		return err
	}

	status.InfoWithBody("Installation details:", indent.String(out, 4))

	return nil
}

// ensureNamespaceCreated creates a k8s namespaces. If it already exists it does nothing.
func ensureNamespaceCreated(ctx context.Context, clientset *kubernetes.Clientset, namespace string) error {
	nsName := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(ctx, nsName, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
