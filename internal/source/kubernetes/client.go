package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Client Kubernetes client
type Client struct {
	dynamicCli   dynamic.Interface
	discoveryCli discovery.DiscoveryInterface
	mapper       meta.RESTMapper
	k8sCli       *kubernetes.Clientset
}

// NewClient initializes Kubernetes client
func NewClient(kubeConfigBytes []byte) (*Client, error) {
	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("while reading kube config. %v", err)
	}
	dynamicCli, discoveryCli, mapper, err := getK8sClients(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("while getting K8s clients. %v", err)
	}
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("while creating K8s clientset. %v", err)
	}
	return &Client{
		dynamicCli:   dynamicCli,
		discoveryCli: discoveryCli,
		k8sCli:       k8sCli,
		mapper:       mapper,
	}, nil
}

func getK8sClients(cfg *rest.Config) (dynamic.Interface, discovery.DiscoveryInterface, meta.RESTMapper, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("while creating discovery client: %w", err)
	}

	dynamicK8sCli, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("while creating dynamic K8s client: %w", err)
	}

	discoCacheClient := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoCacheClient)
	return dynamicK8sCli, discoCacheClient, mapper, nil
}
