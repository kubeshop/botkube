package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// backgroundProcessor is responsible for running background processes.
type backgroundProcessor struct {
	mu          sync.RWMutex
	cancelCtxFn func()
	startTime   time.Time

	errGroup *errgroup.Group
}

// newBackgroundProcessor creates new background processor.
func newBackgroundProcessor() *backgroundProcessor {
	return &backgroundProcessor{}
}

// StartTime returns the start time of the background processor.
func (b *backgroundProcessor) StartTime() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.startTime
}

// Run starts the background processes.
func (b *backgroundProcessor) Run(parentCtx context.Context, fns []func(ctx context.Context)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.startTime = time.Now()
	ctx, cancelFn := context.WithCancel(parentCtx)
	b.cancelCtxFn = cancelFn

	errGroup, errGroupCtx := errgroup.WithContext(ctx)
	b.errGroup = errGroup

	for _, fn := range fns {
		fn := fn
		errGroup.Go(func() error {
			fn(errGroupCtx)
			return nil
		})
	}
}

// StopAndWait stops the background processes and waits for them to finish.
func (b *backgroundProcessor) StopAndWait(log logrus.FieldLogger) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancelCtxFn != nil {
		log.Debug("Cancelling context of the background processor...")
		b.cancelCtxFn()
	}

	if b.errGroup == nil {
		return nil
	}

	log.Debug("Waiting for background processor to finish...")
	return b.errGroup.Wait()
}
