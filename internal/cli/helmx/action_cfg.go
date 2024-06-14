package helmx

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/kubex"
	"github.com/kubeshop/botkube/pkg/ptr"
)

const helmDriver = "secrets"

// GetActionConfiguration returns generic configuration for Helm actions.
func GetActionConfiguration(k8sCfg *kubex.ConfigWithMeta, forNamespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	helmCfg := &genericclioptions.ConfigFlags{
		APIServer:   &k8sCfg.K8s.Host,
		Insecure:    &k8sCfg.K8s.Insecure,
		CAFile:      &k8sCfg.K8s.CAFile,
		BearerToken: &k8sCfg.K8s.BearerToken,
		Context:     &k8sCfg.CurrentContext,
		Namespace:   ptr.FromType(forNamespace),
	}

	debugLog := func(format string, v ...interface{}) {
		if cli.VerboseMode.IsTracing() {
			fmt.Print("    Helm log: ") // if enabled, we need to nest that under Helm step which was already printed with 2 spaces.
			fmt.Printf(format, v...)
			fmt.Println()
		}
	}

	err := actionConfig.Init(helmCfg, forNamespace, helmDriver, debugLog)
	if err != nil {
		return nil, fmt.Errorf("while initializing Helm configuration: %v", err)
	}

	return actionConfig, nil
}
