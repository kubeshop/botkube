package health

type BotkubeStatus string
type PlatformStatusMsg string
type FailureReasonMsg string

const (
	BotkubeStatusHealthy   BotkubeStatus = "Healthy"
	BotkubeStatusUnhealthy BotkubeStatus = "Unhealthy"
)
const (
	StatusUnknown   PlatformStatusMsg = "Unknown"
	StatusHealthy   PlatformStatusMsg = "Healthy"
	StatusUnHealthy PlatformStatusMsg = "Unhealthy"
)

const (
	FailureReasonQuotaExceeded      FailureReasonMsg = "Quota exceeded"
	FailureReasonMaxRetriesExceeded FailureReasonMsg = "Max retries exceeded"
	FailureReasonConnectionError    FailureReasonMsg = "Connection error"
)

// PlatformStatus defines single platform status
type PlatformStatus struct {
	Status   PlatformStatusMsg
	Restarts string
	Reason   FailureReasonMsg
}

// status defines bot agent status.
type status struct {
	Botkube   botStatus                 `json:"botkube"`
	Plugins   map[string]pluginStatuses `json:"plugins,omitempty"`
	Platforms platformStatuses          `json:"platforms,omitempty"`
}

type platformStatuses map[string]PlatformStatus

type pluginStatuses struct {
	Enabled  bool
	Status   string
	Restarts string
}

type botStatus struct {
	Status BotkubeStatus
}
