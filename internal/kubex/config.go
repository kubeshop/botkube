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

// RegisterKubeconfigFlag registers `--kubeconfig` flag.
func RegisterKubeconfigFlag(flags *pflag.FlagSet) {
	flags.StringVar(&kubeconfig, clientcmd.RecommendedConfigPathFlag, "", "Paths to a kubeconfig. Only required if out-of-cluster.")
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
	//  1. --kubeconfig flag
	if kubeconfig != "" {
		c := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}, nil)
		return transform(c)
	}

	// 2. KUBECONFIG environment variable pointing at a file
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) == 0 {
		if c, err := rest.InClusterConfig(); err == nil {
			return &ConfigWithMeta{
				K8s:            c,
				CurrentContext: "In cluster",
			}, nil
		}
	}

	// 3. In-cluster config if running in cluster
	// 4. $HOME/.kube/config if exists
	// 5. user.HomeDir/.kube/config if exists
	//
	// NOTE: For default config file locations, upstream only checks
	// $HOME for the user's home directory, but we can also try
	// os/user.HomeDir when $HOME is unset.
	//
	// TODO(jlanford): could this be done upstream?
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %w", err)
		}
		loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
	}

	return transform(clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil))
}

func transform(c clientcmd.ClientConfig) (*ConfigWithMeta, error) {
	rawConfig, err := c.RawConfig()
	if err != nil {
		return nil, err
	}
	clientConfig, err := c.ClientConfig()
	if err != nil {
		return nil, err
	}
	return &ConfigWithMeta{
		K8s:            clientConfig,
		CurrentContext: rawConfig.CurrentContext,
	}, nil
}
