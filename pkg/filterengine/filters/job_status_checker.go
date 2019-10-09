// JobStatusChecker filter to send notifications only when job succeeds
// and ignore other update events

package filters

import (
	"time"

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

	event.Skip = true
	// Check latest job conditions
	jobLen := len(jobObj.Status.Conditions)
	if jobLen == 0 {
		return
	}
	c := jobObj.Status.Conditions[jobLen-1]
	if c.Type == batchV1.JobComplete {
		event.Messages = []string{"Job succeeded!"}
		// Make sure that we are not considering older events
		// Skip older update events if timestamp difference is more than 30sec
		if !event.TimeStamp.IsZero() && time.Now().Sub(c.LastTransitionTime.Time) > 30*time.Second {
			log.Logger.Debugf("JobStatusChecker Skipping older event: %#v", event)
			return
		}
		event.TimeStamp = c.LastTransitionTime.Time
		// overwrite event.Skip in case of Job succeeded (Job update) events
		event.Skip = false
	}
	event.Reason = c.Reason
	log.Logger.Debug("Job status checker filter successful!")
}

// Describe filter
func (f JobStatusChecker) Describe() string {
	return f.Description
}
