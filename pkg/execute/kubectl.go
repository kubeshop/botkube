package execute

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/utils"
)

const (
	kubectlDisabledMsgFmt            = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
	kubectlNotAuthorizedMsgFmt       = "Sorry, this channel is not authorized to execute kubectl command on cluster '%s'."
	kubectlNotAllowedNamespaceMsgFmt = "Sorry, the kubectl command cannot be executed in %s Namespace on cluster '%s'."
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
// TODO: we can check in general if this is a valid kubectl command
// This will help us to return more friendly message, e.g.
// detect a valid kubectl command but say that you're simply not authorized to use it.
// Additionally, instead of using config with allowed verbs and commands we simply should use related SA.
func (e *Kubectl) CanHandle(args []string) bool {
	if len(args) == 0 {
		return false
	}

	// 1. Check if such kubectl verb is enabled
	if !e.resMapping.AllowedKubectlVerbMap[args[0]] {
		return false
	}

	// 2. Those commands don't have resources specified directly in command. For example:
	//    - kubectl logs foo
	if validDebugCommands[args[0]] {
		return true
	}

	// 3. Check if user has access to a given Kubernetes resource
	// TODO: move to execute..
	if len(args) > 1 && e.matchesAllowedResources(args[1]) { // we can have problems if user will add a flag as a second arg
		return true
	}

	return false
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

// Execute executes kubectl command based on a given args.
func (e *Kubectl) Execute(command string, isAuthChannel bool) (string, error) {
	var (
		// TODO: https://github.com/kubeshop/botkube/issues/596
		// use a related config from communicator bindings.
		kubectlCfg = e.cfg.Executors.GetFirst().Kubectl

		args                 = strings.Fields(strings.TrimSpace(command))
		clusterName          = e.cfg.Settings.ClusterName
		isClusterNamePresent = strings.Contains(command, "--cluster-name")
		inClusterName        = utils.GetClusterNameFromKubectlCmd(command)
		defaultNamespace     = kubectlCfg.DefaultNamespace
	)

	if !kubectlCfg.Enabled {
		if isClusterNamePresent && clusterName == inClusterName {
			return fmt.Sprintf(kubectlDisabledMsgFmt, clusterName), nil
		}
		return "", nil
	}

	if kubectlCfg.RestrictAccess && !isAuthChannel && isClusterNamePresent {
		if clusterName == inClusterName {
			return fmt.Sprintf(kubectlNotAuthorizedMsgFmt, clusterName), nil
		}
		// do not send information, user may have more cluster available on the same channel,
		// and we are probably not the target cluster.
		return "", nil
	}

	args, executionNs, err := e.ensureNamespaceFlag(args, defaultNamespace)
	if err != nil {
		return "", fmt.Errorf("while ensuring Namespace for command execution: %w", err)
	}

	if !utils.IsNamespaceAllowed(kubectlCfg.Namespaces, executionNs) {
		return fmt.Sprintf(kubectlNotAllowedNamespaceMsgFmt, executionNs, clusterName), nil
	}

	finalArgs, shouldExecute := e.getFinalArgs(args, clusterName, isAuthChannel)
	if !shouldExecute {
		return "", nil
	}

	out, err := e.runCmdFn(kubectlBinary, finalArgs)
	if err != nil {
		err = fmt.Errorf("while executing kubectl command: %w", err)
		return fmt.Sprintf("Cluster: %s\n%s%s", clusterName, out, err.Error()), err
	}

	return fmt.Sprintf("Cluster: %s\n%s", clusterName, out), nil
}

// TODO: This code was moved from:
//   https://github.com/kubeshop/botkube/blob/0b99ac480c8e7e93ce721b345ffc54d89019a812/pkg/execute/executor.go#L242-L276
// Further refactoring in needed.
func (e *Kubectl) getFinalArgs(args []string, clusterName string, isAuthChannel bool) ([]string, bool) {
	// Remove unnecessary flags
	var finalArgs []string
	isClusterNameArg := false
	for index, arg := range args {
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
		// Check --cluster-name flag
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			// Check if flag value in current or next argument and compare with config.settings.clusterName
			if arg == ClusterFlag.String() {
				if index == len(args)-1 || e.trimQuotes(args[index+1]) != clusterName {
					return nil, false
				}
				isClusterNameArg = true
			} else {
				if e.trimQuotes(strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1]) != clusterName {
					return nil, false
				}
			}
			isAuthChannel = true
			continue
		}
		finalArgs = append(finalArgs, arg)
	}
	if !isAuthChannel {
		return nil, false
	}

	return finalArgs, true
}

// trim single and double quotes from ends of string
func (e *Kubectl) trimQuotes(clusterValue string) string {
	return strings.TrimFunc(clusterValue, func(r rune) bool {
		if r == unicode.SimpleFold('\u0027') || r == unicode.SimpleFold('\u0022') {
			return true
		}
		return false
	})
}

// getNamespaceFlag returns the namespace value extracted from a given args.
// If `--namespace/-n` was not found, returns 'default'.
func (e *Kubectl) getNamespaceFlag(args []string) (string, error) {
	f := pflag.NewFlagSet("extract-ns", pflag.ContinueOnError)
	var out string
	f.StringVarP(&out, "namespace", "n", "default", "Kubernetes Namespace")
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
