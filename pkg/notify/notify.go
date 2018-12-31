package notify

import (
	"github.com/infracloudio/kubeops/pkg/events"
)

// Notifier to send event notification on the communication channels
type Notifier interface {
	Send(events.Event) error
}
