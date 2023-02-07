package interactive

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
	formatx "github.com/kubeshop/botkube/pkg/format"
)

// RunCommandName defines the button name for the run commands.
const RunCommandName = "Run command"

// HelpMessage provides an option to build the Help message depending on a given platform.
type HelpMessage struct {
	btnBuilder             api.ButtonBuilder
	botName                string
	platform               config.CommPlatformIntegration
	clusterName            string
	enabledPluginExecutors []string
}

// NewHelpMessage return a new instance of HelpMessage.
func NewHelpMessage(platform config.CommPlatformIntegration, clusterName, botName string, executors []string) *HelpMessage {
	btnBuilder := api.ButtonBuilder{BotName: botName}
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
		Base: api.Base{
			Description: fmt.Sprintf("Botkube is now active for %q cluster :rocket:", h.clusterName),
		},
	}

	type getter func() []api.Section
	var sections = []getter{
		h.cluster,
		h.ping,
		h.notificationSections,
		h.actionSections,
		h.configSections,
		h.executorSections,
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

func (h *HelpMessage) cluster() []api.Section {
	switch h.platform {
	case config.SlackCommPlatformIntegration, config.DiscordCommPlatformIntegration, config.MattermostCommPlatformIntegration:
		return []api.Section{
			{
				Base: api.Base{
					Header:      "Using multiple instances",
					Description: fmt.Sprintf("If you are running multiple Botkube instances in the same channel to interact with %s, make sure to specify the cluster name when typing commands.", h.clusterName),
					Body: api.Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%s\n", h.clusterName),
					},
				},
			},
		}
	default:
		return nil
	}
}

func (h *HelpMessage) ping() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header:      "Ping your cluster",
				Description: "Check the status of connected Kubernetes cluster(s).",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("Check status", "ping"),
			},
		},
	}
}

func (h *HelpMessage) filters() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Filters (advanced)",
				Body: api.Body{
					Plaintext: "You can extend Botkube functionality by writing additional filters that can check resource specs, validate some checks and add messages to the Event struct. Learn more at https://docs.botkube.io/filters",
				},
			},
		},
	}
}

func (h *HelpMessage) feedback() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Angry? Amazed?",
			},
			Buttons: []api.Button{
				h.btnBuilder.DescriptionURL("Give feedback", "feedback", "https://feedback.botkube.io", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) footer() []api.Section {
	return []api.Section{
		{
			Buttons: []api.Button{
				h.btnBuilder.ForURL("Read our docs", "https://docs.botkube.io"),
				h.btnBuilder.ForURL("Join our Slack", "https://join.botkube.io"),
				h.btnBuilder.ForURL("Follow us on Twitter", "https://twitter.com/botkube_io"),
			},
		},
	}
}

func (h *HelpMessage) notificationSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Manage incoming notifications",
				Body: api.Body{
					CodeBlock: fmt.Sprintf("%s [enable|disable|status] notifications\n", h.botName),
				},
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Enable notifications", "enable notifications"),
				h.btnBuilder.ForCommandWithoutDesc("Disable notifications", "disable notifications"),
				h.btnBuilder.ForCommandWithoutDesc("Get status", "status notifications"),
			},
		},
		{
			Base: api.Base{
				Header:      "Notification settings for this channel",
				Description: "By default, Botkube will notify only about cluster errors and recommendations.",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("Adjust notifications", "edit SourceBindings", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) actionSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Manage automated actions",
				Body: api.Body{
					CodeBlock: fmt.Sprintf("%s [list|enable|disable] action [action name]\n", h.botName),
				},
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("List available actions", "list actions"),
			},
		},
	}
}

func (h *HelpMessage) configSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "View current Botkube configuration",
				Body: api.Body{
					CodeBlock: fmt.Sprintf("%s show config\n", h.botName),
				},
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Display configuration", "show config"),
			},
		},
	}
}

func (h *HelpMessage) executorSections() []api.Section {
	if h.platform == config.SocketSlackCommPlatformIntegration {
		return []api.Section{
			{
				Base: api.Base{
					Header:      "Interactive kubectl - no typing!",
					Description: "Build kubectl commands interactively",
				},
				Buttons: []api.Button{
					h.btnBuilder.ForCommandWithDescCmd("kubectl", "kubectl", api.ButtonStylePrimary),
				},
			},
			{
				Base: api.Base{
					Description: "To list all enabled executors",
				},
				Buttons: []api.Button{
					h.btnBuilder.ForCommandWithDescCmd("List executors", "list executors"),
				},
			},
			{
				Base: api.Base{
					Description: "To list all command aliases",
				},
				Buttons: []api.Button{
					h.btnBuilder.ForCommandWithDescCmd("List aliases", "list aliases"),
				},
			},
		}
	}

	// without the kubectl command builder
	return []api.Section{
		{
			Base: api.Base{
				Header:      "Run kubectl commands (if enabled)",
				Description: fmt.Sprintf("You can run kubectl commands directly from %s!", formatx.ToTitle(h.platform)),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get services"),
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get pods"),
				h.btnBuilder.ForCommandWithDescCmd(RunCommandName, "kubectl get deployments"),
			},
		},
		{
			Base: api.Base{
				Description: "To list all enabled executors",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("List executors", "list executors"),
			},
		},
		{
			Base: api.Base{
				Description: "To list all command aliases",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("List aliases", "list aliases"),
			},
		},
	}
}

func (h *HelpMessage) pluginHelpSections() []api.Section {
	var out []api.Section
	for _, name := range h.enabledPluginExecutors {
		helpFn, found := pluginHelpProvider[name]
		if !found {
			continue
		}

		platformName := h.platform
		if h.platform == config.SocketSlackCommPlatformIntegration {
			platformName = "slack" // normalize the SocketSlack to Slack
		}

		helpSection := helpFn(formatx.ToTitle(platformName), h.btnBuilder)
		out = append(out, helpSection)
	}
	return out
}
