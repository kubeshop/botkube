package notify

import (
	"fmt"

	"github.com/infracloudio/kubeops/pkg/events"
	log "github.com/infracloudio/kubeops/pkg/logging"
)

type Slack struct {
}

func NewSlack() Notifier {
	return &Slack{}
}

func (s *Slack) Send(event events.Event) error {
	log.Logger.Info(fmt.Sprintf(">>>>>>> Sending to slack: %+v", event))
	return nil
}
