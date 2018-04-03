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
				close(stop)
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
