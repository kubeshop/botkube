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
	Status   PlatformStatusMsg `json:"status,omitempty"`
	Restarts string            `json:"restarts,omitempty"`
	Reason   FailureReasonMsg  `json:"reason,omitempty"`
}

// status defines bot agent status.
type status struct {
	Botkube   botStatus                 `json:"botkube"`
	Plugins   map[string]pluginStatuses `json:"plugins,omitempty"`
	Platforms platformStatuses          `json:"platforms,omitempty"`
}

type platformStatuses map[string]PlatformStatus

type pluginStatuses struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Status   string `json:"status,omitempty"`
	Restarts string `json:"restarts,omitempty"`
}

type botStatus struct {
	Status BotkubeStatus `json:"status,omitempty"`
}
