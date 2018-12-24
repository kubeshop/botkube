package notify

import (
	"github.com/infracloudio/kubeops/pkg/events"
)

type Notifier interface {
	Send(events.Event) error
}
