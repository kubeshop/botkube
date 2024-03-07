package interactive

import (
	"fmt"
	"github.com/kubeshop/botkube/internal/config/remote"
	"golang.org/x/exp/slices"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

// RunCommandName defines the button name for the run commands.
const RunCommandName = "Run command"

// HelpMessage provides an option to build the Help message depending on a given platform.
type HelpMessage struct {
	btnBuilder             *api.ButtonBuilder
	platform               config.CommPlatformIntegration
	clusterName            string
	enabledPluginExecutors []string
}

// NewHelpMessage return a new instance of HelpMessage.
func NewHelpMessage(platform config.CommPlatformIntegration, clusterName string, executors []string) *HelpMessage {
	return &HelpMessage{
		btnBuilder:             api.NewMessageButtonBuilder(),
		platform:               platform,
		clusterName:            clusterName,
		enabledPluginExecutors: executors,
	}
}

// Build returns help message with interactive sections.
func (h *HelpMessage) Build(init bool) CoreMessage {
	msg := CoreMessage{}

	if init {
		msg.Header = fmt.Sprintf(":rocket: Botkube instance %q is now active.", h.clusterName)
	}

	type getter func() []api.Section
	var sections = []getter{
		h.botkubeCloud,
		h.aiPlugin,
		h.basicCommands,
		h.notificationSections,
		//h.actionSections,
		//h.executorSections,
		h.pluginHelpSections,
		h.cluster,
		h.advancedFeatures,
		h.footer,
	}
	for _, add := range sections {
		msg.Sections = append(msg.Sections, add()...)
	}

	return msg
}

func (h *HelpMessage) cluster() []api.Section {
	switch h.platform {
	case config.DiscordCommPlatformIntegration, config.MattermostCommPlatformIntegration:
		return []api.Section{
			{
				Base: api.Base{
					Header:      "Multi-cluster mode",
					Description: "If you have multiple clusters configured for this channel, specify the cluster name when typing commands.",
					Body: api.Body{
						CodeBlock: fmt.Sprintf("--cluster-name=%s\n", h.clusterName),
					},
				},
			},
		}
	case config.CloudSlackCommPlatformIntegration, config.CloudTeamsCommPlatformIntegration:
		return []api.Section{
			{
				Base: api.Base{
					Header:      "🏁 Multi-Cluster flags",
					Description: fmt.Sprintf("`--cluster-name=%q` flag to run a command on this cluster\n", h.clusterName) +
						"`--all-clusters` flag to run commands on all clusters",
				},
			},
		}
	default:
		return nil
	}
}

func (h *HelpMessage) basicCommands() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "🛠️ Basic commands",
				Description: fmt.Sprintf("`%s ping` - ping your cluster and check its status\n", api.MessageBotNamePlaceholder) +
					fmt.Sprintf("`%s list [source|executor|action|alias]` - list available plugins and features", api.MessageBotNamePlaceholder),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Ping cluster", "ping"),
				h.btnBuilder.ForCommandWithoutDesc("List source plugins", "list sources"),
				h.btnBuilder.ForCommandWithoutDesc("List executor plugins", "list executors"),
			},
		},
	}
}

func (h *HelpMessage) footer() []api.Section {
	return []api.Section{
		{
			Buttons: []api.Button{
				h.btnBuilder.ForURL("Give feedback", "https://feedback.botkube.io", api.ButtonStylePrimary),
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
				Header: "📣 Notifications",
				Description: fmt.Sprintf("`%s [enable|disable|status] notifications` - set or query your notification status\n", api.MessageBotNamePlaceholder) +
					fmt.Sprintf("`%s edit sourcebindings` - select notification sources for this channel", api.MessageBotNamePlaceholder),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForURL("Change notification on Cloud", "https://app.botkube.io/instances/9028716b-752b-4e15-88c2-4f8dbf188af7", api.ButtonStylePrimary), // os.Getenv("BOTKUBE_CLOUD_URL") + "/notifications",
				//h.btnBuilder.ForCommandWithoutDesc("Enable", "enable notifications"),
				//h.btnBuilder.ForCommandWithoutDesc("Disable", "disable notifications"),
				//h.btnBuilder.ForCommandWithoutDesc("Get status", "status notifications"),
			},
		},
	}
}

func (h *HelpMessage) actionSections() []api.Section {
	return []api.Section{
		{
			Buttons: []api.Button{
				h.btnBuilder.ForURLWithBoldDesc("Automation help", "Automatically execute commands upon receiving events", "https://docs.botkube.io/usage/automated-actions"),
			},
		},
	}
}

func (h *HelpMessage) executorSections() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Manage executors and aliases",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("List executors", "list executors"),
				h.btnBuilder.ForCommandWithoutDesc("List aliases", "list aliases"),
				h.btnBuilder.ForURL("Executors and aliases help", "https://docs.botkube.io/usage/executor"),
			},
		},
	}
}

func (h *HelpMessage) pluginHelpSections() []api.Section {
	var out []api.Section

	slices.Sort(h.enabledPluginExecutors) // to make the order predictable for testing

	for _, name := range h.enabledPluginExecutors {
		helpFn, found := pluginHelpProvider[name]
		if !found {
			continue
		}

		helpSection := helpFn(h.platform, h.btnBuilder)
		out = append(out, helpSection)
	}
	return out
}

func (h *HelpMessage) botkubeCloud() []api.Section {
	if !remote.IsEnabled() {
		return nil
	}
	return []api.Section{
		{
			Base: api.Base{
				Header: "☁️ Botkube Cloud",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithDescCmd("List connected instances", "cloud list instances"),
				h.btnBuilder.ForCommandWithDescCmd("Set channel default cluster", "cloud set default-instance"),
				h.btnBuilder.ForURL("Open Botkube Cloud", "https://app.botkube.io", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) aiPlugin() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header:      "🤖 AI powered Kubernetes assistant",
				Description: fmt.Sprintf("`%s ai` use natural language to ask any questions", api.MessageBotNamePlaceholder),
			},
			Buttons: []api.Button{
				h.btnBuilder.ForCommandWithoutDesc("Ask a question", "ai hi!", api.ButtonStylePrimary),
			},
		},
	}
}

func (h *HelpMessage) advancedFeatures() []api.Section {
	return []api.Section{
		{
			Base: api.Base{
				Header: "Advanced features",
			},
			Buttons: []api.Button{
				h.btnBuilder.ForURLWithTextDesc("Automation", "Automate your workflows by executing custom commands based on specific events", "https://docs.botkube.io/usage/automated-actions", api.ButtonStylePrimary),
				h.btnBuilder.ForURLWithTextDesc("Aliases", "Run commands like helm, flux, etc. directly from Slack", "https://docs.botkube.io/usage/executor", api.ButtonStylePrimary),
			},
		},
	}

}
