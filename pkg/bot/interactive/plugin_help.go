package interactive

import (
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

type pluginHelpProviderFn func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section

var pluginHelpProvider = map[string]pluginHelpProviderFn{
	"botkube/helm": func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section {
		return api.Section{
			Buttons: []api.Button{
				btnBuilder.ForCommandWithBoldDesc("Helm help", "Run Helm commands", "helm help"),
			},
		}
	},
	"botkube/kubectl": func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section {
		// TODO(https://github.com/kubeshop/botkube-cloud/issues/645): add support for kubectl builder
		if platform.IsInteractive() && platform != config.CloudTeamsCommPlatformIntegration {
			return api.Section{
				Base: api.Base{
					Header: "Run kubectl commands",
				},
				Buttons: []api.Button{
					btnBuilder.ForCommandWithoutDesc("Interactive kubectl", "kubectl", api.ButtonStylePrimary),
					btnBuilder.ForCommandWithoutDesc("kubectl help", "kubectl help"),
				},
			}
		}

		// without the kubectl command builder
		return api.Section{
			Buttons: []api.Button{
				btnBuilder.ForCommandWithBoldDesc("kubectl help", "Run kubectl commands (if enabled)", "kubectl help"),
			},
		}
	},
}
