package kubex

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig string
var kubecontext string

// RegisterKubeconfigFlag registers `--kubeconfig` flag.
func RegisterKubeconfigFlag(flags *pflag.FlagSet) {
	flags.StringVar(&kubeconfig, clientcmd.RecommendedConfigPathFlag, "", "Paths to a kubeconfig. Only required if out-of-cluster.")
	flags.StringVar(&kubecontext, "kubecontext", "", "The name of the kubeconfig context to use.")
}

type ConfigWithMeta struct {
	K8s            *rest.Config
	CurrentContext string
}

// LoadRestConfigWithMetaInformation loads a REST Config. Config precedence:
//
// * --kubeconfig flag pointing at a file
//
// * KUBECONFIG environment variable pointing at a file
//
// * In-cluster config if running in cluster
//
// * $HOME/.kube/config if exists.
//
// code inspired by sigs.k8s.io/controller-runtime@v0.13.1/pkg/client/config/config.go
func LoadRestConfigWithMetaInformation() (*ConfigWithMeta, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	//  1. --kubeconfig flag
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	} else {
		// 2. KUBECONFIG environment variable pointing at a file
		kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
		if len(kubeconfigPath) == 0 {
			// 3. In-cluster config if running in cluster
			if c, err := rest.InClusterConfig(); err == nil {
				return &ConfigWithMeta{
					K8s:            c,
					CurrentContext: "In cluster",
				}, nil
			}
		} else {
			loadingRules.ExplicitPath = kubeconfigPath
			// 4. $HOME/.kube/config if exists
			// 5. user.HomeDir/.kube/config if exists
			//
			// NOTE: For default config file locations, upstream only checks
			// $HOME for the user's home directory, but we can also try
			// os/user.HomeDir when $HOME is unset.
			if _, ok := os.LookupEnv("HOME"); !ok {
				u, err := user.Current()
				if err != nil {
					return nil, fmt.Errorf("could not get current user: %w", err)
				}
				loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
			}
		}
	}

	//  1. --kubecontext flag
	if kubecontext == "" {
		// 2. KUBECONTEXT env
		kubecontext = os.Getenv("KUBECONTEXT")
	}

	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: kubecontext}
	return transform(clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides))
}

func transform(c clientcmd.ClientConfig) (*ConfigWithMeta, error) {
	rawConfig, err := c.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("while getting raw config: %v", err)
	}

	clientConfig, err := c.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("while getting client config: %v", err)
	}

	// 3. load from rawConfig
	if len(kubecontext) == 0 {
		kubecontext = rawConfig.CurrentContext
	}

	return &ConfigWithMeta{
		K8s:            clientConfig,
		CurrentContext: kubecontext,
	}, nil
}
