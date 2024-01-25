package teamsxold

import (
	"fmt"
	"strings"
	"time"

	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
)

const (
	iso8601 = "2006-01-02T15:04:05Z"

	// ReplaceActivityMessageType indicates that the activity should be replaced with a new one.
	ReplaceActivityMessageType = "replaceActivity"
)

// NewRefreshAfterMessage creates a new refresh message for the given id and ttl.
func NewRefreshAfterMessage(id string, ttl time.Time) api.MessageType {
	date := ttl.UTC().Format(iso8601)
	key := fmt.Sprintf("refresh/%s/%s", date, id)

	return api.MessageType(key)
}

// ExtractRefreshMessageMetadata extracts the metadata from the given interactive.CoreMessage.
// Example usage:
//
//	date, cmd, found := ExtractRefreshMessageMetadata(msg)
func ExtractRefreshMessageMetadata(msg interactive.CoreMessage) (string, string, bool) {
	value := string(msg.Type)
	if !strings.HasPrefix(value, "refresh/") {
		return "", "", false
	}
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return "", "", false
	}

	_, err := time.Parse(iso8601, parts[1]) // make sure that date is valid
	if err != nil {
		return "", "", false
	}

	return parts[1], parts[2], true
}

// ActivityForceReplaceOption adds text to the activity.
func ActivityForceReplaceOption() activity.MsgOption {
	return func(activity *schema.Activity) error {
		activity.Type = ReplaceActivityMessageType
		return nil
	}
}
