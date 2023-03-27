package plugin

import (
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	kubeconfigDefaultValue = "default"
)

type KubeConfigInput struct {
	Channel string
}

func GenerateKubeConfig(cfgLoader clientcmd.ClientConfig, pluginCtx config.PluginContext, input KubeConfigInput) ([]byte, error) {
	rbac := pluginCtx.RBAC
	if rbac == nil {
		return nil, nil
	}

	restCfg, err := cfgLoader.ClientConfig()
	if err != nil {
		return nil, err
	}
	rawCfg, err := cfgLoader.RawConfig()
	if err != nil {
		return nil, err
	}

	authInfoName := rawCfg.Contexts[rawCfg.CurrentContext].AuthInfo
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
					Namespace: pluginCtx.DefaultNamespace,
					AuthInfo:  kubeconfigDefaultValue,
				},
			},
		},
		CurrentContext: kubeconfigDefaultValue,
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: kubeconfigDefaultValue,
				AuthInfo: clientcmdapi.AuthInfo{
					Token:                 rawCfg.AuthInfos[authInfoName].Token,
					TokenFile:             rawCfg.AuthInfos[authInfoName].TokenFile,
					ClientCertificateData: rawCfg.AuthInfos[authInfoName].ClientCertificateData,
					ClientKeyData:         rawCfg.AuthInfos[authInfoName].ClientKeyData,
					Impersonate:           generateUserSubject(rbac.User, input),
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

func generateUserSubject(rbac config.UserPolicySubject, input KubeConfigInput) (user string) {
	switch rbac.Type {
	case config.StaticPolicySubjectType:
		user = rbac.Prefix + rbac.Static.Value
	case config.ChannelNamePolicySubjectType:
		user = rbac.Prefix + input.Channel
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
