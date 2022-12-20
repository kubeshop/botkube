package helm

import (
	"fmt"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

func NewActionConfig(k8sCfg *rest.Config, ns string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	helmCfg := &genericclioptions.ConfigFlags{
		APIServer:   &k8sCfg.Host,
		Insecure:    &k8sCfg.Insecure,
		CAFile:      &k8sCfg.CAFile,
		BearerToken: &k8sCfg.BearerToken,
		Namespace:   &ns,
	}

	debugLog := func(format string, v ...interface{}) {
		fmt.Println(fmt.Sprintf(format, v...), zap.String("source", "Helm"))
	}

	err := actionConfig.Init(helmCfg, ns, "", debugLog)
	if err != nil {
		return nil, err
	}
	return actionConfig, nil
}
