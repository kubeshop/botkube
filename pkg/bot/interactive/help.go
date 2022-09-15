package interactive

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/kubeshop/botkube/pkg/config"
)

// Body holds message body fields.
type Body struct {
	CodeBlock string
	Plaintext string
}

// Help represent a help message with interactive sections.
func Help(platform config.CommPlatformIntegration, clusterName, botName string) Message {
	btnBuilder := buttonBuilder{botName: botName}
	return Message{
		Base: Base{
			Description: fmt.Sprintf("BotKube is now active for %q cluster :rocket:", clusterName),
		},
		Sections: []Section{
			{
				Base: Base{
					Header:      "Using multiple instances",
					Description: fmt.Sprintf("If you are running multiple BotKube instances in the same channel to interact with %s, make sure to specify the cluster name when typing commands.", clusterName),
					Body: Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%q", clusterName),
					},
				},
			},
			{
				Base: Base{
					Header:      "Ping your cluster",
					Description: "Check the status of connected Kubernetes cluster(s)",
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd("Check status", "ping"),
				},
			},
			{
				Base: Base{
					Header:      "Run kubectl commands",
					Description: fmt.Sprintf("You can run kubectl commands directly from %s!", cases.Title(language.English).String(string(platform))),
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd("Run commands", "get services"),
					btnBuilder.ForCommandWithDescCmd("Run commands", "get pods"),
					btnBuilder.ForCommandWithDescCmd("Run commands", "get deployments"),
				},
			},
			{
				Base: Base{
					Description: "To list all supported kubectl commands",
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd("List commands", "commands list"),
				},
			},
			{
				Base: Base{
					Header: "Manage incoming notifications",
					Body: Body{
						CodeBlock: fmt.Sprintf("%s notifier [start|stop|status]", botName),
					},
				},
				Buttons: []Button{
					btnBuilder.ForCommand("Start notifications", "notifier start"),
					btnBuilder.ForCommand("Stop notifications", "notifier stop"),
					btnBuilder.ForCommand("Get status", "notifier status"),
				},
			},
			{
				Base: Base{
					Header: "Filters (advanced)",
					Body: Body{
						Plaintext: "You can extend BotKube functionality by writing additional filters that can check resource specs, validate some checks and add messages to the Event struct. Learn more at https://botkube.io/filters",
					},
				},
			},
			{
				Buttons: []Button{
					btnBuilder.ForURL("Read our docs", "https://botkube.io"),
					btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"),
					btnBuilder.ForURL("Follow us on Twitter", "https://twitter.com/botkube_io"),
				},
			},
		},
	}
}
