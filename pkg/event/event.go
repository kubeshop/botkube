package event

import (
	"time"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/pkg/config"
)

// Event stores data about a given event for Kubernetes object.
//
// WARNING: When adding a new field, check if we shouldn't ignore it when marshalling and sending to ELS.
type Event struct {
	metaV1.TypeMeta
	Code            string
	Title           string
	Name            string
	Namespace       string
	Messages        []string
	Type            config.EventType
	Reason          string
	Error           string
	Level           config.Level
	Cluster         string
	Channel         string
	TimeStamp       time.Time
	Count           int32
	Action          string
	Skip            bool `json:",omitempty"`
	Resource        string
	Recommendations []string
	Warnings        []string
	Actions         []Action

	// The following fields are ignored when marshalling the event by purpose.
	// We send the whole Event struct via sink.Elasticsearch integration.
	// When using ELS dynamic mapping, we should avoid complex, dynamic objects, which could result into type conflicts.
	ObjectMeta metaV1.ObjectMeta `json:"-"`
	Object     interface{}       `json:"-"`
}

// Action describes an automated action for a given event.
type Action struct {
	// Command is the command to be executed, with the bot.CrossPlatformBotName prefix.
	Command          string
	ExecutorBindings []string
	DisplayName      string
}

// HasRecommendationsOrWarnings returns true if event has recommendations or warnings.
func (e *Event) HasRecommendationsOrWarnings() bool {
	return len(e.Recommendations) > 0 || len(e.Warnings) > 0
}

// LevelMap is a map of event type to Level
var LevelMap = map[config.EventType]config.Level{
	config.CreateEvent:  config.Info,
	config.UpdateEvent:  config.Warn,
	config.DeleteEvent:  config.Critical,
	config.ErrorEvent:   config.Error,
	config.WarningEvent: config.Error,
}
