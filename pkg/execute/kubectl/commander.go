package kubectl

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
)

// Command defines a command that is executed by the app.
type Command struct {
	Name string
	Cmd  string
}

// Commander is responsible for generating kubectl commands for the given event.
type Commander struct {
	log    logrus.FieldLogger
	merger *Merger
	guard  *CommandGuard
}

// NewCommander creates a new Commander instance.
func NewCommander(log logrus.FieldLogger, merger *Merger, guard *CommandGuard) *Commander {
	return &Commander{log: log, merger: merger, guard: guard}
}

// GetCommandsForEvent returns a list of commands for the given event based on the executor bindings.
func (c *Commander) GetCommandsForEvent(event events.Event, executorBindings []string) ([]Command, error) {
	if event.Type == config.DeleteEvent {
		c.log.Debug("Skipping commands for the DELETE type of event for %q...", event.Kind)
		return nil, nil
	}

	enabledKubectls := c.merger.MergeForNamespace(executorBindings, event.Namespace)

	resourceTypeParts := strings.Split(event.Resource, "/")
	resourceName := resourceTypeParts[len(resourceTypeParts)-1]

	if _, exists := enabledKubectls.AllowedKubectlResource[resourceName]; !exists {
		// resource not allowed
		return nil, nil
	}

	allowedVerbs := enabledKubectls.AllowedKubectlVerb

	resMap, err := c.guard.GetServerResourceMap()
	if err != nil {
		return nil, err
	}

	var commands []Command
	for verb := range allowedVerbs {
		res, err := c.guard.GetResourceDetailsFromMap(verb, resourceName, resMap)
		if err != nil {
			if err == ErrVerbNotFound {
				c.log.Warnf("Not supported verb %q for resource %q. Skipping...", verb, resourceName)
				continue
			}

			return nil, fmt.Errorf("while getting resource details: %w", err)
		}

		var resourceSubstr string
		if res.SlashSeparatedInCommand {
			resourceSubstr = fmt.Sprintf("%s/%s", resourceName, event.Name)
		} else {
			resourceSubstr = fmt.Sprintf("%s %s", resourceName, event.Name)
		}

		var namespaceSubstr string
		if res.Namespaced {
			namespaceSubstr = fmt.Sprintf(" --namespace %s", event.Namespace)
		}

		commands = append(commands, Command{
			Name: verb,
			Cmd:  fmt.Sprintf("%s %s%s", verb, resourceSubstr, namespaceSubstr),
		})
	}

	return commands, nil
}
