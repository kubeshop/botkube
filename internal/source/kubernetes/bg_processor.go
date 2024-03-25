package kubernetes

import (
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

type backgroundProcessor struct {
	mu          sync.RWMutex
	cancelCtxFn func()
	startTime   time.Time

	errGroup *errgroup.Group
}

func newBackgroundProcessor() *backgroundProcessor {
	return &backgroundProcessor{}
}

func (b *backgroundProcessor) StartTime() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.startTime
}

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
