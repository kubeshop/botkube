package uninstall

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/morikuni/aec"

	"github.com/kubeshop/botkube/internal/cli/install/iox"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/cli/uninstall/helm"
	"github.com/kubeshop/botkube/internal/kubex"
)

// Uninstall deletes Botkube Helm release.
func Uninstall(ctx context.Context, w io.Writer, k8sCfg *kubex.ConfigWithMeta, opts Config) (err error) {
	status := printer.NewStatus(w, fmt.Sprintf("Uninstalling %s Helm release...", opts.HelmParams.ReleaseName))
	defer func() {
		status.End(err == nil)
		fmt.Println(aec.Show)
	}()

	err = status.InfoStructFields("Release details:", uninstallationDetails{
		RelName:   opts.HelmParams.ReleaseName,
		Namespace: opts.HelmParams.ReleaseNamespace,
		K8sCtx:    k8sCfg.CurrentContext,
	})
	if err != nil {
		return err
	}

	switch opts.AutoApprove {
	case true:
		status.Infof("Uninstall process will proceed as auto-approval has been explicitly specified")
	case false:
		var confirm bool
		prompt := &survey.Confirm{
			Message: "Do you want to delete existing installation?",
			Default: false,
		}

		questionIndent := iox.NewIndentStdoutWriter("?", 1) // we indent questions by 1 space to match the step layout
		err = survey.AskOne(prompt, &confirm, survey.WithStdio(os.Stdin, questionIndent, os.Stderr))
		if err != nil {
			return fmt.Errorf("while confiriming confirm: %v", err)
		}

		if !confirm {
			status.Infof("Botkube installation not deleted")
			return nil
		}
	}

	uninstaller, err := helm.NewHelm(k8sCfg, opts.HelmParams.ReleaseNamespace)
	if err != nil {
		return err
	}

	return uninstaller.Uninstall(ctx, status, opts.HelmParams)
}

type uninstallationDetails struct {
	// Fields are printed in the same order as defined in struct.
	RelName   string `pretty:"Release Name"`
	Namespace string `pretty:"Namespace"`
	K8sCtx    string `pretty:"Kubernetes Context"`
}
