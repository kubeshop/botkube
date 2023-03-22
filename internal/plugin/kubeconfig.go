package plugin

import (
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	"github.com/kubeshop/botkube/pkg/config"
)

const (
	kubeconfigDefaultValue = "default"
)

func GenerateKubeConfig(restCfg *rest.Config, context config.PluginContext) ([]byte, error) {
	rbac := context.RBAC
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
					Server:               restCfg.Host,
					CertificateAuthority: restCfg.CAFile,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: kubeconfigDefaultValue,
				Context: clientcmdapi.Context{
					Cluster:   kubeconfigDefaultValue,
					Namespace: context.DefaultNamespace,
					AuthInfo:  kubeconfigDefaultValue,
				},
			},
		},
		CurrentContext: kubeconfigDefaultValue,
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: kubeconfigDefaultValue,
				AuthInfo: clientcmdapi.AuthInfo{
					Token:             restCfg.BearerToken,
					TokenFile:         restCfg.BearerTokenFile,
					Impersonate:       generateUserSubject(rbac.User),
					ImpersonateGroups: generateGroupSubject(rbac.Group),
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

func generateUserSubject(rbac config.UserPolicySubject) (user string) {
	switch rbac.Type {
	case config.StaticPolicySubjectType:
		user = rbac.Prefix + rbac.Static.Value
	}
	return
}

func generateGroupSubject(rbac config.GroupPolicySubject) (group []string) {
	switch rbac.Type {
	case config.StaticPolicySubjectType:
		for _, value := range rbac.Static.Values {
			group = append(group, rbac.Prefix+value)
		}
	}
	return
}
