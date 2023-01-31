package alias

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
)

// ListExactForExecutor lists aliases for configured commands that are equal to the executor name.
// For example, for the `botkube/kubectl` executor it lists aliases which are defined for the `kubectl` command only,
// without any additional verbs and flags.
//
// The aliases slice is already sorted alphabetically.
func ListExactForExecutor(rawName string, aliases config.Aliases) []string {
	return listForExecutorWithFn(rawName, aliases, func(cfg config.Alias, executorName string) bool {
		return executorName == cfg.Command
	})
}

// ListForExecutorPrefix lists aliases for configured commands that starts with the executor prefix names.
// For example, for the `botkube/kubectl` executor it lists aliases which are defined for the `kubectl` command,
// and also the `kubectl` commands that contain additional verbs and flags, like `kubectl get pods`.
//
// The aliases slice is already sorted alphabetically.
func ListForExecutorPrefix(rawName string, aliases config.Aliases) []string {
	return listForExecutorWithFn(rawName, aliases, func(cfg config.Alias, executorName string) bool {
		if !strings.HasPrefix(cfg.Command, executorName) {
			return false
		}

		// Case 1: alias equal to executor name
		if len(cfg.Command) == len(executorName) {
			return true
		}

		// Case 2: additional args/flags provided
		executorNameWithSpace := fmt.Sprintf("%s ", executorName)
		return strings.HasPrefix(cfg.Command, executorNameWithSpace)
	})
}

func listForExecutorWithFn(rawName string, aliases config.Aliases, shouldIncludeItem func(cfg config.Alias, executorName string) bool) []string {
	executorName := config.ExecutorNameForKey(rawName)

	var foundAliases []string
	for aliasPrefix, aliasCfg := range aliases {
		if !shouldIncludeItem(aliasCfg, executorName) {
			continue
		}

		foundAliases = append(foundAliases, aliasPrefix)
	}

	sort.Strings(foundAliases)

	return foundAliases
}
