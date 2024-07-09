package heartbeat

import (
	"context"
)

var _ HeartbeatReporter = (*NoopHeartbeatReporter)(nil)

type NoopHeartbeatReporter struct{}

func (n NoopHeartbeatReporter) ReportHeartbeat(context.Context, ReportHeartbeat) error {
	return nil
}
