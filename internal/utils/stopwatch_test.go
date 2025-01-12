package utils_test

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mrsobakin/itmournament/internal/utils"
)

func TestStopwatch_Close(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	var cancelCalled atomic.Bool
	cancelFunc := func() {
		cancelCalled.Store(true)
	}

	sw := utils.NewStopwatch(100*time.Millisecond, cancelFunc)
	sw.Close()
	sw.Resume()
	time.Sleep(200 * time.Millisecond)
	sw.Pause()

	assert.False(t, cancelCalled.Load(), "cancel function should not be called")

	finalGoroutines := runtime.NumGoroutine()

	assert.LessOrEqual(t, finalGoroutines, initialGoroutines, "goroutines should not leak")
}

func TestStopwatch_RepeatClose(t *testing.T) {
	var cancelCalled atomic.Bool
	cancelFunc := func() {
		cancelCalled.Store(true)
	}

	sw := utils.NewStopwatch(100*time.Millisecond, cancelFunc)
	sw.Close()
	assert.NotPanics(t, sw.Close, "Close should not panic on repeated calls")

	assert.False(t, cancelCalled.Load(), "cancel function should be called")
}

func TestStopwatch_DeadlineUpdate(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	var cancelCalled atomic.Bool
	sw := utils.NewStopwatch(100*time.Millisecond, func() {
		cancelCalled.Store(true)
	})

	defer sw.Close()

	go func() {
		time.Sleep(105 * time.Millisecond)
		assert.True(t, cancelCalled.Load(), "cancel function should be called immediately")

		finalGoroutines := runtime.NumGoroutine()

		// Compensate for this goroutine
		assert.LessOrEqual(t, finalGoroutines-1, initialGoroutines, "goroutines should not leak")
	}()

	for i := 0; i < 80; i++ {
		sw.Resume()
		time.Sleep(time.Millisecond)
		sw.Pause()
	}

	sw.Resume()
	time.Sleep(80 * time.Millisecond)
	sw.Pause()

	assert.True(t, cancelCalled.Load(), "cancel function should be called")
}

func TestStopwatchContext_Timeout(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	cause := errors.New("timeout exceeded")
	ctx, sw := utils.NewStopwatchContext(parentCtx, 50*time.Millisecond, cause)
	defer sw.Close()

	go func() {
		time.Sleep(55 * time.Millisecond)
		assert.ErrorIs(t, context.Cause(ctx), cause, "context cancel cause should be the specified cause")

		finalGoroutines := runtime.NumGoroutine()
		// Compensate for this goroutine
		assert.LessOrEqual(t, finalGoroutines-1, initialGoroutines, "goroutines should not leak")
	}()

	sw.Resume()
	time.Sleep(100 * time.Millisecond)
	sw.Pause()
}

func TestStopwatchContext_ParentCancel(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	parentCtx, cancelParent := context.WithCancel(context.Background())

	cause := errors.New("timeout exceeded")
	ctx, sw := utils.NewStopwatchContext(parentCtx, 200*time.Millisecond, cause)
	defer sw.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancelParent()
	}()

	go func() {
		time.Sleep(55 * time.Millisecond)
		assert.NotNil(t, ctx.Err(), "child context should be closed")

		finalGoroutines := runtime.NumGoroutine()
		// Compensate for this goroutine
		assert.LessOrEqual(t, finalGoroutines-1, initialGoroutines, "goroutines should not leak")
	}()

	sw.Resume()
	time.Sleep(100 * time.Millisecond)
	sw.Pause()
}
