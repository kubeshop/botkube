package bot

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

func IsValidNonInteractiveSingleSection(msg interactive.CoreMessage) error {
	if len(msg.Sections) != 1 {
		return fmt.Errorf("event message should contains only one section but got %d", len(msg.Sections))
	}
	if msg.Type != api.NonInteractiveSingleSection {
		return fmt.Errorf("this renderer is limited to single section message, cannot be used for type %s", msg.Type)
	}

	return nil
}
