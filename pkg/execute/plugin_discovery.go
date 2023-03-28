package execute

var staticPluginDiscovery = map[string]string{
	"kubectl": "No `kubectl` commands are enabled in this channel. To learn how to enable them, visit https://docs.botkube.io/configuration/executor/kubectl",
	"helm":    "No `helm` commands are enabled in this channel. To learn how to enable them, visit https://docs.botkube.io/configuration/executor/helm",
}

// GetInstallHelpForKnownPlugin returns install help for a known plugin.
func GetInstallHelpForKnownPlugin(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}

	cmdName := args[0]
	help, found := staticPluginDiscovery[cmdName]
	return help, found
}
