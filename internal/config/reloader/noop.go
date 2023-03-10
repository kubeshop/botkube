package reloader

import "context"

var _ Reloader = (*NoopReloader)(nil)

// NoopReloader is a reloader that does nothing.
type NoopReloader struct{}

// NewNoopReloader returns new NoopReloader.
func NewNoopReloader() *NoopReloader {
	return &NoopReloader{}
}

// Do does nothing.
func (u *NoopReloader) Do(context.Context) error {
	return nil
}
