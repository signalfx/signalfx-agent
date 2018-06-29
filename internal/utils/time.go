package utils

import (
	"context"
	"sync"
	"time"
)

// Debounce0 calls a zero arg function on the trailing edge of every `duration`.
func Debounce0(fn func(), duration time.Duration) (func(), chan<- struct{}) {
	lock := &sync.Mutex{}

	stop := make(chan struct{})
	timer := time.NewTicker(duration)
	callRequested := false

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-timer.C:
				if callRequested {
					lock.Lock()

					fn()
					callRequested = false

					lock.Unlock()
				}
			}
		}
	}()

	return func() {
		lock.Lock()
		callRequested = true
		lock.Unlock()
	}, stop
}

// RunOnInterval the given fn once every interval, starting at the moment the
// function is called.  Returns a function that can be called to stop running
// the function.
func RunOnInterval(ctx context.Context, fn func(), interval time.Duration) {
	timer := time.NewTicker(interval)

	fn()
	go func() {
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				fn()
			}
		}
	}()
}

// RepeatPolicy repeat behavior for RunOnIntervals Function
type RepeatPolicy int

const (
	// RepeatAll repeats all intervals
	RepeatAll RepeatPolicy = iota
	// RepeatLast repeats only the last interval
	RepeatLast
	// RepeatNone does not repeat
	RepeatNone
)

// RunOnIntervals the given function once on the specified intervals, and
// repeat according to the supplied RepeatPolicy.
func RunOnIntervals(ctx context.Context, fn func(), intervals []time.Duration, repeatPolicy RepeatPolicy) {
	if len(intervals) < 1 {
		return
	}
	// copy intervals
	intv := intervals[:]

	// initialize timer
	timer := time.NewTimer(intv[0])
	go func() {
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:

				if len(intv) > 1 {
					// advance the interval
					intv = intv[1:]
				} else {
					// evaluate repeat policies
					if repeatPolicy == RepeatNone {
						return
					} else if repeatPolicy == RepeatAll {
						intv = intervals[:] // copy the original interval list
					}
				}
				timer.Reset(intervals[0])
				fn()
			}
		}
	}()
}
