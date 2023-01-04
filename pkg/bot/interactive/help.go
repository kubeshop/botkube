package interactive

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/kubeshop/botkube/pkg/config"
)

// RunCommandName defines the button name for the run commands.
const RunCommandName = "Run command"

// HelpMessage provides an option to build the Help message depending on a given platform.
type HelpMessage struct {
	btnBuilder             ButtonBuilder
	botName                string
	platform               config.CommPlatformIntegration
	clusterName            string
	enabledPluginExecutors []string
}

// NewHelpMessage return a new instance of HelpMessage.
func NewHelpMessage(platform config.CommPlatformIntegration, clusterName, botName string, executors []string) *HelpMessage {
	btnBuilder := ButtonBuilder{BotName: botName}
	return &HelpMessage{
		btnBuilder:             btnBuilder,
		botName:                botName,
		platform:               platform,
		clusterName:            clusterName,
		enabledPluginExecutors: executors,
	}
}

// Build returns help message with interactive sections.
func (h *HelpMessage) Build() Message {
	msg := Message{
		Base: Base{
			Description: fmt.Sprintf("Botkube is now active for %q cluster :rocket:", h.clusterName),
		},
	}

	type getter func() []Section
	var sections = []getter{
		h.cluster,
		h.ping,
		h.notificationSections,
		h.actionSections,
		h.configSections,
		h.kubectlSections,
		h.pluginHelpSections,
		h.filters,
		h.feedback,
		h.footer,
	}
	for _, add := range sections {
		msg.Sections = append(msg.Sections, add()...)
	}

	return msg
}

func (h *HelpMessage) cluster() []Section {
	switch h.platform {
	case config.SlackCommPlatformIntegration, config.DiscordCommPlatformIntegration, config.MattermostCommPlatformIntegration:
		return []Section{
			{
				Base: Base{
					Header:      "Using multiple instances",
					Description: fmt.Sprintf("If you are running multiple Botkube instances in the same channel to interact with %s, make sure to specify the cluster name when typing commands.", h.clusterName),
					Body: Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%s\n", h.clusterName),
					},
				},
			},
		}
	default:
		return nil
	}
}

func (h *HelpMessage) ping() []Section {
	return []Section{
		{
			Base: Base{
				Header:      "Ping your cluster",
				Description: "Check the status of connected Kubernetes cluster(s).",
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithDescCmd("Check status", "ping"),
			},
		},
	}
}

func (h *HelpMessage) filters() []Section {
	return []Section{
		{
			Base: Base{
				Header: "Filters (advanced)",
				Body: Body{
					Plaintext: "You can extend Botkube functionality by writing additional filters that can check resource specs, validate some checks and add messages to the Event struct. Learn more at https://docs.botkube.io/filters",
				},
			},
		},
	}
}

func (h *HelpMessage) feedback() []Section {
	return []Section{
		{
			Base: Base{
				Header: "Angry? Amazed?",
			},
			Buttons: []Button{
				h.btnBuilder.DescriptionURL("Give feedback", "feedback", "https://feedback.botkube.io", ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) footer() []Section {
	return []Section{
		{
			Buttons: []Button{
				h.btnBuilder.ForURL("Read our docs", "https://docs.botkube.io"),
				h.btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"),
				h.btnBuilder.ForURL("Follow us on Twitter", "https://twitter.com/botkube_io"),
			},
		},
	}
}

func (h *HelpMessage) notificationSections() []Section {
	return []Section{
		{
			Base: Base{
				Header: "Manage incoming notifications",
				Body: Body{
					CodeBlock: fmt.Sprintf("%s [start|stop|status] notifications\n", h.botName),
				},
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithoutDesc("Start notifications", "start notifications"),
				h.btnBuilder.ForCommandWithoutDesc("Stop notifications", "stop notifications"),
				h.btnBuilder.ForCommandWithoutDesc("Get status", "status notifications"),
			},
		},
		{
			Base: Base{
				Header:      "Notification settings for this channel",
				Description: "By default, Botkube will notify only about cluster errors and recommendations.",
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithDescCmd("Adjust notifications", "edit SourceBindings", ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) actionSections() []Section {
	return []Section{
		{
			Base: Base{
				Header: "Manage automated actions",
				Body: Body{
					CodeBlock: fmt.Sprintf("%s [list|enable|disable] action [action name]\n", h.botName),
				},
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithoutDesc("List available actions", "list actions"),
			},
		},
	}
}

func (h *HelpMessage) configSections() []Section {
	return []Section{
		{
			Base: Base{
				Header: "View current Botkube configuration",
				Body: Body{
					CodeBlock: fmt.Sprintf("%s config\n", h.botName),
				},
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithoutDesc("Display configuration", "config"),
			},
		},
	}
}

func (h *HelpMessage) kubectlSections() []Section {
	// TODO(https://github.com/kubeshop/botkube/issues/802): remove this warning in after releasing 0.17.
	warn := ":warning: Botkube 0.17 and above require a prefix (`k`, `kc`, `kubectl`) when running kubectl commands through the bot.\n\ne.g. `@Botkube k get pods` instead of `@Botkube get pods`\n"

	if h.platform == config.SocketSlackCommPlatformIntegration {
		return []Section{
			{
				Base: Base{
					Header:      "Interactive kubectl - no typing!",
					Description: warn,
				},
			},
			{
				Base: Base{
					Description: "Build kubectl commands interactively",
				},
				Buttons: []Button{
					h.btnBuilder.ForCommandWithDescCmd("kubectl", "kubectl", ButtonStylePrimary),
				},
				Context: ContextItems{
					{
						Text: "Alternatively use kubectl as usual with all supported commands\n" +
							"`k | kc | kubectl [verb] [resource] [flags]`",
					},
				},
			},
			{
				Base: Base{
					Description: "To list all enabled executors",
				},
				Buttons: []Button{
					h.btnBuilder.ForCommandWithDescCmd("List executors", "list executors"),
				},
			},
		}
	}

	// without the kubectl command builder
	return []Section{
		{
			Base: Base{
				Header:      "Run kubectl commands (if enabled)",
				Description: fmt.Sprintf("%s\nYou can run kubectl commands directly from %s!", warn, cases.Title(language.English).String(string(h.platform))),
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get services"),
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get pods"),
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get deployments"),
			},
		},
		{
			Base: Base{
				Description: "To list all enabled executors",
			},
			Buttons: []Button{
				h.btnBuilder.ForCommandWithDescCmd("List executors", "list executors"),
			},
		},
	}
}

func (h *HelpMessage) pluginHelpSections() []Section {
	var out []Section
	for _, name := range h.enabledPluginExecutors {
		helpFn, found := pluginHelpProvider[name]
		if !found {
			continue
		}

		platformName := h.platform
		if h.platform == config.SocketSlackCommPlatformIntegration {
			platformName = "slack" // normalize the SocketSlack to Slack
		}

		helpSection := helpFn(cases.Title(language.English).String(string(platformName)), h.btnBuilder)
		out = append(out, helpSection)
	}
	return out
}
