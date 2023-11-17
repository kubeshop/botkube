package analytics

import (
	"time"

	"github.com/kubeshop/botkube/internal/analytics/batched"
)

func (r *SegmentReporter) SetIdentity(identity *Identity) {
	r.identity = identity
}

func (r *SegmentReporter) Identity() *Identity {
	return r.identity
}

func (r *SegmentReporter) SetBatchedData(batchedData BatchedDataStore) {
	r.batchedData = batchedData
}

func (r *SegmentReporter) SetTickDuration(tickDuration time.Duration) {
	r.tickDuration = tickDuration
}

func (r *SegmentReporter) ReportHeartbeatEvent() error {
	return r.reportHeartbeatEvent()
}

func (r *SegmentReporter) HeartbeatProperties() batched.HeartbeatProperties {
	return r.batchedData.HeartbeatProperties()
}
