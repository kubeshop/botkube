package bot

import (
	"context"
	"errors"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
)

func TestWithRetriesFunc(t *testing.T) {
	t.Run("Stop immediately on Unrecoverable error", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fixErr := errors.New("some random error")
		bot := &CloudSlack{}
		// when
		retriesFinalError := make(chan error, 1)
		go func() {
			retriesFinalError <- bot.withRetries(ctx, loggerx.NewNoop(), 5, func() error {
				return retry.Unrecoverable(fixErr)
			})
			close(retriesFinalError)
		}()

		// then
		awaitExpectations(t, 500*time.Millisecond, func() {
			gotErr := <-retriesFinalError
			assert.ErrorIs(t, gotErr, fixErr)
		})
	})

	t.Run("Continues retries on recoverable errors", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fixErr := errors.New("some random error")
		bot := &CloudSlack{}
		// when
		retriesFinalError := make(chan error, 1)
		go func() {
			retriesFinalError <- bot.withRetries(ctx, loggerx.NewNoop(), 5, func() error {
				return fixErr
			})
			close(retriesFinalError)
		}()

		// then
		assert.Never(t, func() bool {
			<-retriesFinalError
			return true
		}, time.Second, 10*time.Millisecond)
	})

	t.Run("Respected max retries", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fixErr := errors.New("some random error")
		bot := &CloudSlack{}
		// when
		retriesFinalError := make(chan error, 1)
		go func() {
			retriesFinalError <- bot.withRetries(ctx, loggerx.NewNoop(), 0, func() error {
				return fixErr
			})
			close(retriesFinalError)
		}()

		// then
		awaitExpectations(t, time.Second, func() {
			gotErr := <-retriesFinalError
			assert.ErrorIs(t, gotErr, fixErr)
		})
	})

	t.Run("Respected canceled context", func(t *testing.T) {
		// given
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		fixErr := errors.New("some random error")
		bot := &CloudSlack{}
		// when
		retriesFinalError := make(chan error, 1)
		go func() {
			retriesFinalError <- bot.withRetries(canceledCtx, loggerx.NewNoop(), 5, func() error {
				return retry.Unrecoverable(fixErr)
			})
			close(retriesFinalError)
		}()

		// then
		awaitExpectations(t, time.Second, func() {
			gotErr := <-retriesFinalError
			assert.ErrorIs(t, gotErr, canceledCtx.Err())
		})
	})
}

func awaitExpectations(t *testing.T, dur time.Duration, assertion func()) {
	t.Helper()

	finished := make(chan struct{})
	go func() {
		assertion()
		close(finished)
	}()

	select {
	case <-time.After(dur):
		t.Fatalf("expected function was not fulfilled within given %s duration", dur)
	case <-finished:
		return
	}
}
