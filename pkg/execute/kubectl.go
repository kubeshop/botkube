package execute

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/utils"
)

const (
	kubectlNotAuthorizedMsgFmt          = "Sorry, this channel is not authorized to execute kubectl command on cluster '%s'."
	kubectlNotAllowedNamespaceMsgFmt    = "Sorry, the kubectl command cannot be executed in the '%s' Namespace on cluster '%s'. Use 'commands list' to see all allowed namespaces."
	kubectlNotAllowedAllNamespaceMsgFmt = "Sorry, the kubectl command cannot be executed in all Namespace on cluster '%s'. Use 'commands list' to see all allowed namespaces."
	kubectlNotAllowedKindMsgFmt         = "Sorry, the kubectl command is not authorized to work with '%s' resources on cluster '%s'. Use 'commands list' to see all allowed resources."
	kubectlFlagAfterVerbMsg             = "Please specify the resource name after the verb, and all flags after the resource name. Format <verb> <resource> [flags]"
	kubectlDefaultNamespace             = "default"
)

// Kubectl executes kubectl commands using local binary.
type Kubectl struct {
	log logrus.FieldLogger
	cfg config.Config

	kcChecker *kubectl.Checker
	runCmdFn  CommandRunnerFunc
	merger    *kubectl.Merger
}

// NewKubectl creates a new instance of Kubectl.
func NewKubectl(log logrus.FieldLogger, cfg config.Config, merger *kubectl.Merger, kcChecker *kubectl.Checker, fn CommandRunnerFunc) *Kubectl {
	return &Kubectl{
		log:       log,
		cfg:       cfg,
		merger:    merger,
		kcChecker: kcChecker,
		runCmdFn:  fn,
	}
}

// CanHandle returns true if it's allowed kubectl command that can be handled by this executor.
//
// TODO: we should just introduce a command name explicitly. In this case `@BotKube kubectl get po` instead of `@BotKube get po`
// As a result, we are able to detect kubectl command but say that you're simply not authorized to use it instead of "Command not supported. (..)"
func (e *Kubectl) CanHandle(bindings []string, args []string) bool {
	if len(args) == 0 {
		return false
	}

	// Check if such kubectl verb is enabled
	if !e.kcChecker.IsKnownVerb(e.merger.MergeAllEnabledVerbs(bindings), args[0]) {
		return false
	}

	return true
}

// Execute executes kubectl command based on a given args.
//
// This method should be called ONLY if:
// - we are a target cluster,
// - and Kubectl.CanHandle returned true.
func (e *Kubectl) Execute(bindings []string, command string, isAuthChannel bool) (string, error) {
	log := e.log.WithFields(logrus.Fields{
		"isAuthChannel": isAuthChannel,
		"command":       command,
	})

	log.Debugf("Handling command...")

	var (
		args        = strings.Fields(strings.TrimSpace(command))
		clusterName = e.cfg.Settings.ClusterName
		verb        = args[0]
		resource    = e.getResourceName(args)
	)

	executionNs, err := e.getCommandNamespace(args)
	if err != nil {
		return "", fmt.Errorf("while extracting Namespace from command: %w", err)
	}
	if executionNs == "" { // namespace not found in command, so find default and add `-n` flag to args
		executionNs = e.findDefaultNamespace(bindings)
		args = e.addNamespaceFlag(args, executionNs)
	}

	kcConfig := e.merger.MergeForNamespace(bindings, executionNs)

	if !isAuthChannel && kcConfig.RestrictAccess {
		msg := fmt.Sprintf(kubectlNotAuthorizedMsgFmt, clusterName)
		return e.omitIfIfWeAreNotExplicitlyTargetCluster(log, command, msg)
	}

	if !e.kcChecker.IsVerbAllowedInNs(kcConfig, verb) {
		if executionNs == config.AllNamespaceIndicator {
			return fmt.Sprintf(kubectlNotAllowedAllNamespaceMsgFmt, clusterName), nil
		}
		return fmt.Sprintf(kubectlNotAllowedNamespaceMsgFmt, executionNs, clusterName), nil
	}

	// Some commands don't have resources specified directly in command. For example:
	// - kubectl logs foo
	if !validDebugCommands[verb] && resource != "" {
		if !e.validResourceName(resource) {
			return kubectlFlagAfterVerbMsg, nil
		}
		// Check if user has access to a given Kubernetes resource
		// TODO: instead of using config with allowed verbs and commands we simply should use related SA.
		if !e.kcChecker.IsResourceAllowedInNs(kcConfig, resource) {
			return fmt.Sprintf(kubectlNotAllowedKindMsgFmt, resource, clusterName), nil
		}
	}

	finalArgs := e.getFinalArgs(args)
	out, err := e.runCmdFn(kubectlBinary, finalArgs)
	if err != nil {
		return fmt.Sprintf("Cluster: %s\n%s%s", clusterName, out, err.Error()), nil
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
// If `--namespace/-n` was not found, returns empty string.
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

// getAllNamespaceFlag returns the namespace value extracted from a given args.
// If `--A, --all-namespaces` was not found, returns empty string.
func (e *Kubectl) getAllNamespaceFlag(args []string) (bool, error) {
	f := pflag.NewFlagSet("extract-ns", pflag.ContinueOnError)
	// ignore unknown flags errors, e.g. `--cluster-name` etc.
	f.ParseErrorsWhitelist.UnknownFlags = true

	var out bool
	f.BoolVarP(&out, "all-namespaces", "A", false, "Kubernetes All Namespaces")
	if err := f.Parse(args); err != nil {
		return false, err
	}
	return out, nil
}

func (e *Kubectl) getCommandNamespace(args []string) (string, error) {
	// 1. Check for `-A, --all-namespaces` in args. Based on the kubectl manual:
	//    "Namespace in current context is ignored even if specified with --namespace."
	inAllNs, err := e.getAllNamespaceFlag(args)
	if err != nil {
		return "", err
	}
	if inAllNs {
		return config.AllNamespaceIndicator, nil // TODO: find all namespaces
	}

	// 2. Check for `-n/--namespace` in args
	executionNs, err := e.getNamespaceFlag(args)
	if err != nil {
		return "", err
	}
	if executionNs != "" {
		return executionNs, nil
	}

	return "", nil
}

func (e *Kubectl) findDefaultNamespace(bindings []string) string {
	// 1. Merge all enabled kubectls, to find the defaultNamespace settings
	cfg := e.merger.MergeAllEnabled(bindings)
	if cfg.DefaultNamespace != "" {
		// 2. Use user defined default
		return cfg.DefaultNamespace
	}

	// 3. If not found, explicitly use `default` namespace.
	return kubectlDefaultNamespace
}

// addNamespaceFlag add namespace to returned args list.
func (e *Kubectl) addNamespaceFlag(args []string, defaultNamespace string) []string {
	return append([]string{"-n", defaultNamespace}, utils.DeleteDoubleWhiteSpace(args)...)
}

func (e *Kubectl) getResourceName(args []string) string {
	if len(args) < 2 {
		return ""
	}
	parts := strings.SplitN(args[1], "/", 2)
	resource := parts[0]
	return resource
}

func (e *Kubectl) validResourceName(resource string) bool {
	// ensures that resource name starts with letter
	return unicode.IsLetter(rune(resource[0]))
}
