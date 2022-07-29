package execute

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/utils"
)

const (
	kubectlDisabledMsgFmt            = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
	kubectlNotAuthorizedMsgFmt       = "Sorry, this channel is not authorized to execute kubectl command on cluster '%s'."
	kubectlNotAllowedNamespaceMsgFmt = "Sorry, the kubectl command cannot be executed in the '%s' Namespace on cluster '%s'. Use 'commands list' to see all allowed namespaces."
	kubectlNotAllowedKindMsgFmt      = "Sorry, the kubectl command is not authorized to work with '%s' resources on cluster '%s'. Use 'commands list' to see all allowed resources."
	kubectlDefaultNamespace          = "default"
)

// Kubectl executes kubectl commands using local binary.
type Kubectl struct {
	log        logrus.FieldLogger
	cfg        config.Config
	resMapping ResourceMapping
	runCmdFn   CommandRunnerFunc
}

// NewKubectl creates a new instance of Kubectl.
func NewKubectl(log logrus.FieldLogger, cfg config.Config, mapping ResourceMapping, fn CommandRunnerFunc) *Kubectl {
	return &Kubectl{
		log:        log,
		cfg:        cfg,
		resMapping: mapping,
		runCmdFn:   fn,
	}
}

// CanHandle returns true if it's allowed kubectl command that can be handled by this executor.
//
// TODO: we should just introduce a command name explicitly. In this case `@BotKube kubectl get po` instead of `@BotKube get po`
// As a result, we are able to detect kubectl command but say that you're simply not authorized to use it instead of "Command not supported. (..)"
func (e *Kubectl) CanHandle(args []string) bool {
	if len(args) == 0 {
		return false
	}

	// Check if such kubectl verb is enabled
	if !e.resMapping.AllowedKubectlVerbMap[args[0]] {
		return false
	}

	return true
}

// Execute executes kubectl command based on a given args.
//
// This method should be called ONLY if:
// - we are a target cluster,
// - and Kubectl.CanHandle returned true.
func (e *Kubectl) Execute(command string, isAuthChannel bool) (string, error) {
	log := e.log.WithFields(logrus.Fields{
		"isAuthChannel": isAuthChannel,
		"command":       command,
	})

	log.Debugf("Handling command...")

	var (
		// TODO: https://github.com/kubeshop/botkube/issues/596
		// use a related config from communicator bindings.
		kubectlCfg = e.cfg.Executors.GetFirst().Kubectl

		args             = strings.Fields(strings.TrimSpace(command))
		clusterName      = e.cfg.Settings.ClusterName
		defaultNamespace = kubectlCfg.DefaultNamespace
	)

	if !isAuthChannel && kubectlCfg.RestrictAccess {
		msg := fmt.Sprintf(kubectlNotAuthorizedMsgFmt, clusterName)
		return e.omitIfIfWeAreNotExplicitlyTargetCluster(log, command, msg)
	}

	if !kubectlCfg.Enabled {
		msg := fmt.Sprintf(kubectlDisabledMsgFmt, clusterName)
		return e.omitIfIfWeAreNotExplicitlyTargetCluster(log, command, msg)
	}

	// Some commands don't have resources specified directly in command. For example:
	// - kubectl logs foo
	if !validDebugCommands[args[0]] {
		// Check if user has access to a given Kubernetes resource
		// TODO: instead of using config with allowed verbs and commands we simply should use related SA.
		if len(args) > 1 && !e.matchesAllowedResources(args[1]) {
			return fmt.Sprintf(kubectlNotAllowedKindMsgFmt, args[1], clusterName), nil
		}
	}

	args, executionNs, err := e.ensureNamespaceFlag(args, defaultNamespace)
	if err != nil {
		return "", fmt.Errorf("while ensuring Namespace for command execution: %w", err)
	}

	if !kubectlCfg.Namespaces.IsAllowed(executionNs) {
		return fmt.Sprintf(kubectlNotAllowedNamespaceMsgFmt, executionNs, clusterName), nil
	}

	finalArgs := e.getFinalArgs(args)
	out, err := e.runCmdFn(kubectlBinary, finalArgs)
	if err != nil {
		errCtx := fmt.Errorf("while executing kubectl command: %w", err)
		return fmt.Sprintf("Cluster: %s\n%s%s", clusterName, out, err.Error()), errCtx
	}

	return fmt.Sprintf("Cluster: %s\n%s", clusterName, out), nil
}

// omitIfIfWeAreNotExplicitlyTargetCluster returns verboseMsg if there is explicit '--cluster-name' flag that matches this cluster.
// It's useful if we want to be more verbose, but we also don't want to spam if we are not the target one.
func (e *Kubectl) omitIfIfWeAreNotExplicitlyTargetCluster(log *logrus.Entry, cmd string, verboseMsg string) (string, error) {
	if utils.GetClusterNameFromKubectlCmd(cmd) == e.cfg.Settings.ClusterName {
		return verboseMsg, nil
	}

	log.WithField("verboseMsg", verboseMsg).Debugf("Skipping kubectl verbose message...")
	return "", nil
}

// TODO: This code was moved from:
//   https://github.com/kubeshop/botkube/blob/0b99ac480c8e7e93ce721b345ffc54d89019a812/pkg/execute/executor.go#L242-L276
// Further refactoring in needed. For example, the cluster flag should be removed by an upper layer as it's strictly
// as it's strictly BotKube related and not executor specific (e.g. kubectl, helm, istio etc.)
func (e *Kubectl) getFinalArgs(args []string) []string {
	// Remove unnecessary flags
	var finalArgs []string
	isClusterNameArg := false
	for _, arg := range args {
		if isClusterNameArg {
			isClusterNameArg = false
			continue
		}
		if arg == AbbrFollowFlag.String() || strings.HasPrefix(arg, FollowFlag.String()) {
			continue
		}
		if arg == AbbrWatchFlag.String() || strings.HasPrefix(arg, WatchFlag.String()) {
			continue
		}
		// Remove --cluster-name flag and it's value
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			// Check if flag value in current or next argument and compare with config.settings.clusterName
			if arg == ClusterFlag.String() {
				isClusterNameArg = true
			}
			continue
		}
		finalArgs = append(finalArgs, arg)
	}

	return finalArgs
}

// getNamespaceFlag returns the namespace value extracted from a given args.
// If `--namespace/-n` was not found, returns 'default'.
func (e *Kubectl) getNamespaceFlag(args []string) (string, error) {
	f := pflag.NewFlagSet("extract-ns", pflag.ContinueOnError)
	// ignore unknown flags errors, e.g. `--cluster-name` etc.
	f.ParseErrorsWhitelist.UnknownFlags = true

	var out string
	f.StringVarP(&out, "namespace", "n", "", "Kubernetes Namespace")
	if err := f.Parse(args); err != nil {
		return "", err
	}
	return out, nil
}

// ensureNamespaceFlag ensures that a Namespace flag is available in args. If necessary, add it to returned args list.
func (e *Kubectl) ensureNamespaceFlag(args []string, defaultNamespace string) ([]string, string, error) {
	executionNs, err := e.getNamespaceFlag(args)
	if err != nil {
		return nil, "", fmt.Errorf("while getting Namespace for command execution: %w", err)
	}
	if executionNs != "" { // was specified in a received command
		return args, executionNs, nil
	}

	if defaultNamespace == "" {
		defaultNamespace = kubectlDefaultNamespace
	}

	args = append([]string{"-n", defaultNamespace}, utils.DeleteDoubleWhiteSpace(args)...)

	return args, defaultNamespace, nil
}

func (e *Kubectl) matchesAllowedResources(name string) bool {
	variants := []string{
		// received name
		name,
		// normalized short name
		e.resMapping.ShortnameResourceMap[strings.ToLower(name)],
		// normalized kind name
		e.resMapping.KindResourceMap[strings.ToLower(name)],
	}

	for _, name := range variants {
		if e.resMapping.AllowedKubectlResourceMap[name] {
			return true
		}
	}

	return false
}
