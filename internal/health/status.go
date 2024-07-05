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
	ErrorMsg string            `json:"errorMsg,omitempty"`
}

// Status defines bot agent status.
type Status struct {
	Botkube   BotStatus               `json:"botkube"`
	Plugins   map[string]PluginStatus `json:"plugins,omitempty"`
	Platforms platformStatuses        `json:"platforms,omitempty"`
}

type platformStatuses map[string]PlatformStatus

type PluginStatus struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Status   string `json:"status,omitempty"`
	Restarts string `json:"restarts,omitempty"`
}

type BotStatus struct {
	Status BotkubeStatus `json:"status,omitempty"`
}
