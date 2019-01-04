package notify

import (
	"github.com/infracloudio/botkube/pkg/events"
)

// Notifier to send event notification on the communication channels
type Notifier interface {
	SendEvent(events.Event) error
	SendMessage(string) error
}
