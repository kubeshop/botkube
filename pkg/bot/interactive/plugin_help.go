package interactive

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
)

type pluginHelpProviderFn func(platform string, btnBuilder *api.ButtonBuilder) api.Section

var pluginHelpProvider = map[string]pluginHelpProviderFn{
	"botkube/helm": func(platform string, btnBuilder *api.ButtonBuilder) api.Section {
		return api.Section{
			Base: api.Base{
				Header:      "Run Helm commands",
				Description: fmt.Sprintf("You can run Helm commands directly from %s!", platform),
			},
			Buttons: []api.Button{
				btnBuilder.ForCommandWithDescCmd("Show help", "helm help"),
			},
		}
	},
}
