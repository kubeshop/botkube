// JobStatusChecker filter to send notifications only when job succeeds
// and ignore other update events

package filters

import (
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	log "github.com/infracloudio/botkube/pkg/logging"

	batchV1 "k8s.io/api/batch/v1"
)

// JobStatusChecker checks job status and adds message in the events structure
type JobStatusChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(JobStatusChecker{
		Description: "Sends notifications only when job succeeds and ignores other job update events.",
	})
}

// Run filers and modifies event struct
func (f JobStatusChecker) Run(object interface{}, event *events.Event) {
	// Run filter only on Job update event
	if event.Kind != "Job" || event.Type != config.UpdateEvent {
		return
	}
	jobObj, ok := object.(*batchV1.Job)
	if !ok {
		return
	}

	// Check latest job conditions
	jobLen := len(jobObj.Status.Conditions)
	if jobLen == 0 {
		event.Skip = true
		return
	}
	c := jobObj.Status.Conditions[jobLen-1]
	if c.Type == batchV1.JobComplete {
		event.Messages = []string{"Job succeeded!"}
		event.TimeStamp = c.LastTransitionTime.Time
	} else {
		event.Skip = true
		return
	}
	event.Reason = c.Reason
	log.Logger.Debug("Job status checker filter successful!")
}

// Describe filter
func (f JobStatusChecker) Describe() string {
	return f.Description
}
