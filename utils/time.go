package utils

import (
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
