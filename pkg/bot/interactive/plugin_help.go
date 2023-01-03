package interactive

import "fmt"

type pluginHelpProviderFn func(platform string, btnBuilder ButtonBuilder) Section

var pluginHelpProvider = map[string]pluginHelpProviderFn{
	"botkube/helm": func(platform string, btnBuilder ButtonBuilder) Section {
		return Section{
			Base: Base{
				Header:      "Run Helm commands",
				Description: fmt.Sprintf("You can run Helm commands directly from %s!", platform),
			},
			Buttons: []Button{
				btnBuilder.ForCommandWithDescCmd("Show help", "helm help"),
			},
		}
	},
}
