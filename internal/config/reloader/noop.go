package reloader

import "context"

var _ Reloader = (*NoopReloader)(nil)

type NoopReloader struct{}

func NewNoopReloader() *NoopReloader {
	return &NoopReloader{}
}

func (u *NoopReloader) Do(context.Context) error {
	return nil
}
