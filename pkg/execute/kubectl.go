package execute

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gookit/color"
	"github.com/mattn/go-shellwords"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

const (
	// KubectlBinary is absolute path of kubectl binary
	KubectlBinary = "/usr/local/bin/kubectl"
)

const (
	kubectlNotAuthorizedMsgFmt         = "Sorry, this channel is not authorized to execute kubectl command on cluster '%s'."
	kubectlNotAllowedVerbMsgFmt        = "Sorry, the kubectl '%s' command cannot be executed in the '%s' Namespace on cluster '%s'. Use 'list executors' to see allowed executors."
	kubectlNotAllowedVerbInAllNsMsgFmt = "Sorry, the kubectl '%s' command cannot be executed for all Namespaces on cluster '%s'. Use 'list executors' to see allowed executors."
	kubectlNotAllowedKindMsgFmt        = "Sorry, the kubectl command is not authorized to work with '%s' resources in the '%s' Namespace on cluster '%s'. Use 'list executors' to see allowed executors."
	kubectlNotAllowedKinInAllNsMsgFmt  = "Sorry, the kubectl command is not authorized to work with '%s' resources for all Namespaces on cluster '%s'. Use 'list executors' to see allowed executors."
	kubectlFlagAfterVerbMsg            = "Please specify the resource name after the verb, and all flags after the resource name. Format <verb> <resource> [flags]"
	kubectlDefaultNamespace            = "default"
)

// resourcelessCommands holds all commands that don't specify resources directly. For example:
// - kubectl logs foo
// - kubectl cluster-info
var resourcelessCommands = map[string]struct{}{
	"exec":          {},
	"logs":          {},
	"attach":        {},
	"auth":          {},
	"api-versions":  {},
	"cluster-info":  {},
	"cordon":        {},
	"drain":         {},
	"uncordon":      {},
	"run":           {},
	"api-resources": {},
	"rollout":       {},
}

// Kubectl executes kubectl commands using local binary.
type Kubectl struct {
	log logrus.FieldLogger
	cfg config.Config

	kcChecker *kubectl.Checker
	cmdRunner CommandCombinedOutputRunner
	merger    *kubectl.Merger
}

// NewKubectl creates a new instance of Kubectl.
func NewKubectl(log logrus.FieldLogger, cfg config.Config, merger *kubectl.Merger, kcChecker *kubectl.Checker, fn CommandCombinedOutputRunner) *Kubectl {
	return &Kubectl{
		log:       log,
		cfg:       cfg,
		merger:    merger,
		kcChecker: kcChecker,
		cmdRunner: fn,
	}
}

// CanHandle returns true if it's allowed kubectl command that can be handled by this executor.
func (e *Kubectl) CanHandle(args []string) bool {
	if len(args) == 0 {
		return false
	}

	// make sure that verb is also specified
	// empty `k|kc|kubectl` commands are handled by command builder
	return len(args) >= 2 && args[0] == kubectlCommandName
}

// GetCommandPrefix gets verb command with k8s alias prefix.
func (e *Kubectl) GetCommandPrefix(args []string) string {
	if len(args) < 2 {
		return ""
	}

	return fmt.Sprintf("%s %s", args[0], args[1])
}

// getArgsWithoutAlias gets command without k8s alias.
func (e *Kubectl) getArgsWithoutAlias(msg string) ([]string, error) {
	msgParts, err := shellwords.Parse(strings.TrimSpace(msg))
	if err != nil {
		return nil, fmt.Errorf("while parsing the command message into args: %w", err)
	}

	if len(msgParts) >= 2 && msgParts[0] == kubectlCommandName {
		return msgParts[1:], nil
	}

	return msgParts, nil
}

// Execute executes kubectl command based on a given args.
//
// This method should be called ONLY if:
// - we are a target cluster,
// - and Kubectl.CanHandle returned true.
func (e *Kubectl) Execute(bindings []string, command string, isAuthChannel bool, cmdCtx CommandContext) (string, error) {
	log := e.log.WithFields(logrus.Fields{
		"isAuthChannel": isAuthChannel,
		"command":       command,
	})

	log.Debugf("Handling command...")

	args, err := e.getArgsWithoutAlias(command)
	if err != nil {
		return "", err
	}

	var (
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
		msg := NewExecutionCommandError(kubectlNotAuthorizedMsgFmt, clusterName)
		return "", e.omitIfWeAreNotExplicitlyTargetCluster(log, msg, cmdCtx)
	}

	if !e.kcChecker.IsVerbAllowedInNs(kcConfig, verb) {
		if executionNs == config.AllNamespaceIndicator {
			return "", NewExecutionCommandError(kubectlNotAllowedVerbInAllNsMsgFmt, verb, clusterName)
		}
		return "", NewExecutionCommandError(kubectlNotAllowedVerbMsgFmt, verb, executionNs, clusterName)
	}

	_, isResourceless := resourcelessCommands[verb]
	if !isResourceless && resource != "" {
		if !e.validResourceName(resource) {
			return "", NewExecutionCommandError(kubectlFlagAfterVerbMsg)
		}
		// Check if user has access to a given Kubernetes resource
		// TODO: instead of using config with allowed verbs and commands we simply should use related SA.
		if !e.kcChecker.IsResourceAllowedInNs(kcConfig, resource) {
			if executionNs == config.AllNamespaceIndicator {
				return "", NewExecutionCommandError(kubectlNotAllowedKinInAllNsMsgFmt, resource, clusterName)
			}
			return "", NewExecutionCommandError(kubectlNotAllowedKindMsgFmt, resource, executionNs, clusterName)
		}
	}

	finalArgs := e.getFinalArgs(args)
	out, err := e.cmdRunner.RunCombinedOutput(KubectlBinary, finalArgs)
	out = color.ClearCode(out)
	if err != nil {
		return "", NewExecutionCommandError("%s%s", out, err.Error())
	}

	return out, nil
}

// omitIfWeAreNotExplicitlyTargetCluster returns verboseMsg if there is explicit '--cluster-name' flag that matches this cluster.
// It's useful if we want to be more verbose, but we also don't want to spam if we are not the target one.
func (e *Kubectl) omitIfWeAreNotExplicitlyTargetCluster(log *logrus.Entry, verboseMsg *ExecutionCommandError, cmdCtx CommandContext) error {
	if cmdCtx.ProvidedClusterNameEqual() {
		return verboseMsg
	}

	log.WithField("verboseMsg", verboseMsg).Debugf("Skipping kubectl verbose message...")
	return nil
}

// TODO: This code was moved from:
//
//	https://github.com/kubeshop/botkube/blob/0b99ac480c8e7e93ce721b345ffc54d89019a812/pkg/execute/executor.go#L242-L276
//
// Further refactoring in needed. For example, the cluster flag should be removed by an upper layer
// as it's strictly Botkube related and not executor specific (e.g. kubectl, helm, istio etc.).
func (e *Kubectl) getFinalArgs(args []string) []string {
	// Remove unnecessary flags
	var finalArgs []string
	for _, arg := range args {
		if arg == AbbrFollowFlag.String() || strings.HasPrefix(arg, FollowFlag.String()) {
			continue
		}
		if arg == AbbrWatchFlag.String() || strings.HasPrefix(arg, WatchFlag.String()) {
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
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")

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
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")

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
	return append([]string{"-n", defaultNamespace}, sliceutil.FilterEmptyStrings(args)...)
}

func (e *Kubectl) getResourceName(args []string) string {
	if len(args) < 2 {
		return ""
	}
	resource, _, _ := strings.Cut(args[1], "/")
	return resource
}

func (e *Kubectl) validResourceName(resource string) bool {
	// ensures that resource name starts with letter
	return unicode.IsLetter(rune(resource[0]))
}
