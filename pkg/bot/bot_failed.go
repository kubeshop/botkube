package bot

import (
	"github.com/kubeshop/botkube/internal/health"
)

type HealthNotifierBot interface {
	GetStatus() health.PlatformStatus
}

// FailedBot mock of bot, uses for healthChecker.
type FailedBot struct {
	status        health.PlatformStatusMsg
	failureReason health.FailureReasonMsg
	errorMsg      string
}

// NewBotFailed creates a new FailedBot instance.
func NewBotFailed(failureReason health.FailureReasonMsg, errorMsg string) *FailedBot {
	return &FailedBot{
		status:        health.StatusUnHealthy,
		failureReason: failureReason,
		errorMsg:      errorMsg,
	}
}

// GetStatus gets bot status.
func (b *FailedBot) GetStatus() health.PlatformStatus {
	return health.PlatformStatus{
		Status:   b.status,
		Restarts: "0/0",
		Reason:   b.failureReason,
		ErrorMsg: b.errorMsg,
	}
}
