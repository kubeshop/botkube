package kubex

import (
	"net"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
)

// BuildConfigFromFlags modified after copied from here https://github.com/kubernetes/client-go/blob/cf830e3cb3abbcc32cc1b6bea4feb1a9a1881af3/tools/clientcmd/client_config.go#L616
func BuildConfigFromFlags(masterUrl, kubeconfigPath, saTokenPath string) (*rest.Config, error) {
	if kubeconfigPath == "" && masterUrl == "" {
		klog.Warning("Neither --kubeconfig nor --master was specified.  Using the inClusterConfig.  This might not work.")
		kubeconfig, err := inClusterConfig(saTokenPath)
		if err == nil {
			return kubeconfig, nil
		}
		kubeconfig.AuthProvider = &clientcmdapi.AuthProviderConfig{
			Name: "token-file",
			Config: map[string]string{
				"token-file": saTokenPath,
			},
		}
		klog.Warning("error creating inClusterConfig, falling back to default config: ", err)
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterUrl}}).ClientConfig()
}

// inClusterConfig is copied from https://github.com/kubernetes/client-go/blob/cf830e3cb3abbcc32cc1b6bea4feb1a9a1881af3/rest/config.go#L511
func inClusterConfig(saTokenPrefix string) (*rest.Config, error) {
	tokenFile := saTokenPrefix + "-token"
	rootCAFile := saTokenPrefix + "-ca.crt"
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, rest.ErrNotInCluster
	}

	// #nosec G304
	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	tlsClientConfig := rest.TLSClientConfig{}

	if _, err := certutil.NewPool(rootCAFile); err != nil {
		klog.Errorf("Expected to load root CA config from %s, but got err: %v", rootCAFile, err)
	} else {
		tlsClientConfig.CAFile = rootCAFile
	}

	return &rest.Config{
		// TODO: switch to using cluster DNS.
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(token),
		BearerTokenFile: tokenFile,
	}, nil
}
