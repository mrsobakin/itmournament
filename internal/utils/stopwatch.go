package utils

import (
	"context"
	"sync/atomic"
	"time"
)

// Keeps track of the time that passes between `Stopwatch.Resume()`
// and `Stopwatch.Pause()` calls.
//
// If at some point while the stopwatch is running, it's summary running
// time exceedes the given timeout, provided `cancelFunc` is called.
//
// To avoid having dangling goroutines, `Stopwatch.Close()` should be
// called when the stopwatch is no longer needed.
//
// Stopwatch is not thread safe.
type Stopwatch struct {
	totalPassed     time.Duration
	timeout         time.Duration
	lastResume      time.Time
	deadlineUpdates chan<- *time.Duration
	closed          atomic.Bool
}

// Creates Stopwatch with given timeout and cancelFunc.
//
// Created Stopwatch is in PAUSED state.
func NewStopwatch(timeout time.Duration, cancelFunc func()) *Stopwatch {
	deadlineUpdates := make(chan *time.Duration)

	s := Stopwatch{
		totalPassed:     0,
		timeout:         timeout,
		lastResume:      time.Time{},
		deadlineUpdates: deadlineUpdates,
		closed:          atomic.Bool{},
	}

	go func(deadlineUpdates <-chan *time.Duration) {
		var timeLeft *time.Duration = nil

		for {
			if timeLeft != nil {
				select {
				case <-time.After(*timeLeft):
					{
						s.Close()
						cancelFunc()
						return
					}
				case newTimeLeft, ok := <-deadlineUpdates:
					{
						if !ok {
							return
						}
						timeLeft = newTimeLeft
					}
				}
			} else {
				newTimeLeft, ok := <-deadlineUpdates
				if !ok {
					return
				}
				timeLeft = newTimeLeft
			}
		}
	}(deadlineUpdates)

	return &s
}

func (s *Stopwatch) Resume() {
	s.lastResume = time.Now()

	untilDeadline := s.timeout - s.totalPassed
	if !s.closed.Load() {
		s.deadlineUpdates <- &untilDeadline
	}
}

func (s *Stopwatch) Pause() {
	addDuration := time.Since(s.lastResume)
	if !s.closed.Load() {
		// TODO: race here: message is not yet sended, but the channel has just closed.
		s.deadlineUpdates <- nil
	}
	s.totalPassed += addDuration
}

func (s *Stopwatch) Close() {
	if !s.closed.Swap(true) {
		close(s.deadlineUpdates)
	}
}

// Creates context and stopwatch bounded together.
//
// When stopwatch summary exceedes `timeout`, context is cancelled
// with `cause` cause.
//
// When the parent context is cancelled, stopwatch is closed.
func NewStopwatchContext(parent context.Context, timeout time.Duration, cause error) (context.Context, *Stopwatch) {
	ctx, cancel := context.WithCancelCause(parent)
	sw := NewStopwatch(timeout, func() {
		cancel(cause)
	})

	go func() {
		select {
		case <-parent.Done():
		case <-ctx.Done():
		}
		sw.Close()
	}()

	return ctx, sw
}
