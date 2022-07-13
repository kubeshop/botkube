package analytics

func (r *SegmentReporter) SetIdentity(identity *Identity) {
	r.identity = identity
}

func (r *SegmentReporter) Identity() *Identity {
	return r.identity
}
