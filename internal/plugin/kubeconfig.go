package plugin

import (
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	kubeconfigDefaultValue     = "default"
	kubeconfigDefaultNamespace = "default"
)

type KubeConfigInput struct {
	Channel string
}

func GenerateKubeConfig(restCfg *rest.Config, pluginCtx config.PluginContext, input KubeConfigInput) ([]byte, error) {
	rbac := pluginCtx.RBAC
	if rbac == nil {
		return nil, nil
	}
	apiCfg := clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: kubeconfigDefaultValue,
				Cluster: clientcmdapi.Cluster{
					Server:                   restCfg.Host,
					CertificateAuthority:     restCfg.CAFile,
					CertificateAuthorityData: restCfg.CAData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: kubeconfigDefaultValue,
				Context: clientcmdapi.Context{
					Cluster:   kubeconfigDefaultValue,
					Namespace: kubeconfigDefaultNamespace,
					AuthInfo:  kubeconfigDefaultValue,
				},
			},
		},
		CurrentContext: kubeconfigDefaultValue,
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: kubeconfigDefaultValue,
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
			user = "botkube-internal-static-user"
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
