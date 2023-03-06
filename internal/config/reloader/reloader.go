package reloader

import "context"

// Reloader is an interface for reloading configuration.
type Reloader interface {
	Do(ctx context.Context) error
}

// ResourceVersionHolder is an interface for holding resource version with ability to set it.
type ResourceVersionHolder interface {
	SetResourceVersion(int)
}
