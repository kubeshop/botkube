package filterengine

import "github.com/infracloudio/botkube/pkg/filterengine/filters"

var (
	// DefaultFilterEngine contains default implementation for FilterEngine
	DefaultFilterEngine FilterEngine
)

// SetupGlobal set ups a global filter engine with all default filters enabled
// TODO: Convert it to local instance shared by all BotKube components
func SetupGlobal() {
	DefaultFilterEngine = NewDefaultFilter()
	DefaultFilterEngine.RegisterMany([]Filter{
		filters.ImageTagChecker{Description: "Checks and adds recommendation if 'latest' image tag is used for container image."},
		filters.IngressValidator{Description: "Checks if services and tls secrets used in ingress specs are available."},
		filters.ObjectAnnotationChecker{Description: "Checks if annotations botkube.io/* present in object specs and filters them."},
		filters.PodLabelChecker{Description: "Checks and adds recommendations if labels are missing in the pod specs."},
		filters.NamespaceChecker{Description: "Checks if event belongs to blocklisted namespaces and filter them."},
		filters.NodeEventsChecker{Description: "Sends notifications on node level critical events."},
	})
}
