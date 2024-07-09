package health

// Failed represents failed platform.
type Failed struct {
	status        PlatformStatusMsg
	failureReason FailureReasonMsg
	errorMsg      string
}

// NewFailed creates a new Failed instance.
func NewFailed(failureReason FailureReasonMsg, errorMsg string) *Failed {
	return &Failed{
		status:        StatusUnHealthy,
		failureReason: failureReason,
		errorMsg:      errorMsg,
	}
}

// GetStatus gets bot status.
func (b *Failed) GetStatus() PlatformStatus {
	return PlatformStatus{
		Status:   b.status,
		Restarts: "0/0",
		Reason:   b.failureReason,
		ErrorMsg: b.errorMsg,
	}
}
