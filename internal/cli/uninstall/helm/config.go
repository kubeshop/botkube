package helm

import (
	"time"
)

// Config holds Helm configuration parameters.
type Config struct {
	ReleaseName      string
	ReleaseNamespace string

	DisableHooks        bool
	DryRun              bool
	KeepHistory         bool
	Wait                bool
	DeletionPropagation string
	Timeout             time.Duration
	Description         string
}
