package filters

import (
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

var IgnoredReasons = []string{
	"DeadlineExceeded",
	"FailedScheduling",
}

// ReasonChecker will reject events if reason are prohibited
type ReasonChecker struct {
	Description string
}

// Register filter
func init() {
	filterengine.DefaultFilterEngine.Register(ReasonChecker{
		Description: "Ignore event if reasons are listed in IgnoredReasons.",
	})
}

// Run filters and modifies event struct
func (f ReasonChecker) Run(object interface{}, event *events.Event) {
	// Check for Event object
	if utils.GetObjectTypeMetaData(object).Kind != "Event" {
		return
	}

	for _, v := range IgnoredReasons {
		if v == event.Reason {
			log.Debug("Event with ignored reason is filtered!")
			event.Skip = true
			break
		}
	}
}

// Describe filter
func (f ReasonChecker) Describe() string {
	return f.Description
}
