package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	kubeconfigDefaultValue     = "default"
	kubeconfigDefaultNamespace = "default"
)

// KubeConfigInput defines the input for GenerateKubeConfig.
type KubeConfigInput struct {
	Channel string
}

// GenerateKubeConfig generates kubeconfig based on RBAC policy.
func GenerateKubeConfig(restCfg *rest.Config, clusterName string, pluginCtx config.PluginContext, input KubeConfigInput) ([]byte, error) {
	if clusterName == "" {
		clusterName = kubeconfigDefaultValue
	}

	rbac := pluginCtx.RBAC
	if rbac == nil {
		return nil, nil
	}

	if rbac.User.Type == config.EmptyPolicySubjectType && rbac.Group.Type == config.EmptyPolicySubjectType {
		// that means the Kubeconfig shouldn't be generated
		return nil, nil
	}

	apiCfg := clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmdapi.Cluster{
					Server:                   restCfg.Host,
					CertificateAuthority:     restCfg.CAFile,
					CertificateAuthorityData: restCfg.CAData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: clusterName,
				Context: clientcmdapi.Context{
					Cluster:   clusterName,
					Namespace: kubeconfigDefaultNamespace,
					AuthInfo:  clusterName,
				},
			},
		},
		CurrentContext: clusterName,
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: clusterName,
				AuthInfo: clientcmdapi.AuthInfo{
					Token:                 restCfg.BearerToken,
					TokenFile:             restCfg.BearerTokenFile,
					ClientCertificateData: restCfg.CertData,
					ClientKeyData:         restCfg.KeyData,
					Impersonate:           generateUserSubject(rbac.User, rbac.Group, input),
					ImpersonateGroups:     generateGroupSubject(rbac.Group, input),
				},
			},
		},
	}

	yamlKubeConfig, err := yaml.Marshal(apiCfg)
	if err != nil {
		return nil, err
	}

	return yamlKubeConfig, nil
}

func generateUserSubject(rbac config.UserPolicySubject, group config.GroupPolicySubject, input KubeConfigInput) (user string) {
	switch rbac.Type {
	case config.StaticPolicySubjectType:
		user = rbac.Prefix + rbac.Static.Value
	case config.ChannelNamePolicySubjectType:
		user = rbac.Prefix + input.Channel
	default:
		if group.Type != config.EmptyPolicySubjectType {
			user = config.RBACDefaultUser
		}
	}
	return
}

func generateGroupSubject(rbac config.GroupPolicySubject, input KubeConfigInput) (group []string) {
	switch rbac.Type {
	case config.StaticPolicySubjectType:
		for _, value := range rbac.Static.Values {
			group = append(group, rbac.Prefix+value)
		}
	case config.ChannelNamePolicySubjectType:
		group = append(group, rbac.Prefix+input.Channel)
	}
	return
}

// PersistKubeConfig creates a temporary kubeconfig file and returns its path and a function to delete it.
func PersistKubeConfig(_ context.Context, kc []byte) (string, func(context.Context) error, error) {
	if len(kc) == 0 {
		return "", nil, fmt.Errorf("received empty kube config")
	}

	file, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}
	defer file.Close()

	abs, err := filepath.Abs(file.Name())
	if err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}

	if _, err = file.Write(kc); err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}

	deleteFn := func(context.Context) error {
		return os.RemoveAll(abs)
	}

	return abs, deleteFn, nil
}

// ValidateKubeConfigProvided returns an error if a given kubeconfig is empty or nil.
func ValidateKubeConfigProvided(pluginName string, kubeconfig []byte) error {
	if len(kubeconfig) != 0 {
		return nil
	}
	return fmt.Errorf("The kubeconfig data is missing. Please make sure that you have specified a valid RBAC configuration for %q plugin. Learn more at https://docs.botkube.io/configuration/rbac.", pluginName)
}
