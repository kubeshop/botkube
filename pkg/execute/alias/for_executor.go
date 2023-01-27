package alias

import (
	"github.com/kubeshop/botkube/pkg/config"
	"sort"
)

func ListForExecutor(rawName string, aliases config.Aliases) []string {
	executorName := config.ExecutorNameForKey(rawName)

	var foundAliases []string
	for aliasPrefix, aliasCfg := range aliases {
		if executorName != aliasCfg.Command {
			continue
		}

		foundAliases = append(foundAliases, aliasPrefix)
	}

	sort.Strings(foundAliases)

	return foundAliases
}
