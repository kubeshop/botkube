package alias

import (
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
)

// ExpandPrefix expands alias prefix with the target command based on provided configuration.
// The function requires already sanitized input raw command - no whitespace characters at the beginning or end are allowed.
func ExpandPrefix(rawCmd string, aliases config.Aliases) string {
	for aliasPrefix, aliasCfg := range aliases {
		if !strings.HasPrefix(rawCmd, aliasPrefix) {
			continue
		}

		// Case 1: just an alias provided
		if len(rawCmd) == len(aliasPrefix) {
			return aliasCfg.Command
		}

		// Case 2: Additional args/flags provided
		aliasWithSpace := fmt.Sprintf("%s ", aliasPrefix)
		if strings.HasPrefix(rawCmd, aliasWithSpace) {
			targetCmdWithSpace := fmt.Sprintf("%s ", aliasCfg.Command)
			return strings.Replace(rawCmd, aliasWithSpace, targetCmdWithSpace, 1)
		}

		// Case 3: False positive - alias prefix is a part of the command - continue
	}

	return rawCmd
}
