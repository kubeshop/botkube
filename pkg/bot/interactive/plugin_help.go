package interactive

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/formatx"
)

type pluginHelpProviderFn func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section

var pluginHelpProvider = map[string]pluginHelpProviderFn{
	"botkube/helm": func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section {
		return api.Section{
			Base: api.Base{
				Header:      "Run Helm commands",
				Description: fmt.Sprintf("You can run Helm commands directly from %s!", platformDisplayName(platform)),
			},
			Buttons: []api.Button{
				btnBuilder.ForCommandWithDescCmd("Show help", "helm help"),
			},
		}
	},
	"botkube/kubectl": func(platform config.CommPlatformIntegration, btnBuilder *api.ButtonBuilder) api.Section {
		if platform.IsInteractive() {
			return api.Section{
				Base: api.Base{
					Header:      "Interactive kubectl - no typing!",
					Description: "Build kubectl commands interactively",
				},
				Buttons: []api.Button{
					btnBuilder.ForCommandWithDescCmd("kubectl", "kubectl", api.ButtonStylePrimary),
				},
			}
		}

		// without the kubectl command builder
		return api.Section{
			Base: api.Base{
				Header:      "Run kubectl commands (if enabled)",
				Description: fmt.Sprintf("You can run kubectl commands directly from %s!", platformDisplayName(platform)),
			},
			Buttons: []api.Button{
				btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get services"),
				btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get pods"),
				btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get deployments"),
			},
		}
	},
}

func platformDisplayName(platform config.CommPlatformIntegration) string {
	platformName := platform
	if platform == config.SocketSlackCommPlatformIntegration {
		platformName = "slack" // normalize the SocketSlack to Slack
	}
	return formatx.ToTitle(platformName)
}
