package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestWithRetriesFunc(t *testing.T) {
	t.Run("Stop immediately on Unrecoverable error", func(t *testing.T) {
		// given
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		fixErr := errors.New("some random error")
		bot, err := NewCloudSlack(loggerx.NewNoop(), "", config.CloudSlack{}, "clusterName", nil, nil)
		require.NoError(t, err)
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
		bot, err := NewCloudSlack(loggerx.NewNoop(), "", config.CloudSlack{}, "clusterName", nil, nil)
		require.NoError(t, err)
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
		bot, err := NewCloudSlack(loggerx.NewNoop(), "", config.CloudSlack{}, "clusterName", nil, nil)
		require.NoError(t, err)
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
		bot, err := NewCloudSlack(loggerx.NewNoop(), "", config.CloudSlack{}, "clusterName", nil, nil)
		require.NoError(t, err)
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
