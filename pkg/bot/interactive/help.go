package interactive

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/kubeshop/botkube/pkg/config"
)

// RunCommandName defines the button name for the run commands.
const RunCommandName = "Run command"

// Help represent a help message with interactive sections.
func Help(platform config.CommPlatformIntegration, clusterName, botName string) Message {
	btnBuilder := ButtonBuilder{BotName: botName}
	return Message{
		Base: Base{
			Description: fmt.Sprintf("Botkube is now active for %q cluster :rocket:", clusterName),
		},
		Sections: []Section{
			{
				Base: Base{
					Header:      "Using multiple instances",
					Description: fmt.Sprintf("If you are running multiple Botkube instances in the same channel to interact with %s, make sure to specify the cluster name when typing commands.", clusterName),
					Body: Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%s\n", clusterName),
					},
				},
			},
			{
				Base: Base{
					Header: "Manage incoming notifications",
					Body: Body{
						CodeBlock: fmt.Sprintf("%s notifier [start|stop|status]\n", botName),
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
					Header:      "Notification settings for this channel",
					Description: "By default, Botkube will notify only about cluster errors and recommendations.",
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd("Adjust notifications", "edit SourceBindings", ButtonStylePrimary),
				},
			},
			{
				Base: Base{
					Header:      "Ping your cluster",
					Description: "Check the status of connected Kubernetes cluster(s).",
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd("Check status", "ping"),
				},
			},
			{
				Base: Base{
					Header:      "Run kubectl commands (if enabled)",
					Description: fmt.Sprintf("You can run kubectl commands directly from %s!", cases.Title(language.English).String(string(platform))),
				},
				Buttons: []Button{
					btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get services"),
					btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get pods"),
					btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get deployments"),
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
					Header: "Filters (advanced)",
					Body: Body{
						Plaintext: "You can extend Botkube functionality by writing additional filters that can check resource specs, validate some checks and add messages to the Event struct. Learn more at https://botkube.io/filters",
					},
				},
			},
			{
				Base: Base{
					Header: "Angry? Amazed?",
				},
				Buttons: []Button{
					btnBuilder.DescriptionURL("Give feedback", "feedback", "https://feedback.botkube.io", ButtonStylePrimary),
				},
			},
			{
				Buttons: []Button{
					btnBuilder.ForURL("Read our docs", "https://botkube.io/docs"),
					btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"),
					btnBuilder.ForURL("Follow us on Twitter", "https://twitter.com/botkube_io"),
				},
			},
		},
	}
}
