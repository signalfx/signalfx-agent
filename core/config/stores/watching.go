package stores

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/libkv/store"
)

const filesystemScheme = "file"

// reconnectTime is time between store reconnect attempts
var reconnectTime = 30 * time.Second

const (
	// WatchInitial is the initial watch state
	WatchInitial = iota
	// WatchFailed is the watch failed state
	WatchFailed
	// Watching is the normal watching state
	Watching
)

type source interface {
	AtomicPut(key string, value []byte, previous *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error)
	Close()
	Exists(string) (bool, error)
	Get(string) (*store.KVPair, error)
	List(directory string) ([]*store.KVPair, error)
	Put(key string, value []byte, opts *store.WriteOptions) error
	Watch(string, <-chan struct{}) (<-chan *store.KVPair, error)
	WatchTree(string, <-chan struct{}) (<-chan []*store.KVPair, error)
}

// EnsureExists creates an empty file if it doesn't already exist
func EnsureExists(src source, path string) error {
	exists, err := src.Exists(path)
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("creating empty file %s", path)
		if _, _, err := src.AtomicPut(path, nil, nil, nil); err != nil {
			return err
		}
	}

	return nil
}

// WatchRobust returns a reliable channel that hides the underlying libkv
// watch because it will fail when files go away
func WatchRobust(source source, path string, stop <-chan struct{}) (<-chan *store.KVPair, error) {
	retCh := make(chan *store.KVPair)

	go func() {
		state := WatchInitial
		var stopCh chan struct{}
		var ch <-chan *store.KVPair
		var err error

		for {
			switch state {
			case WatchFailed:
				// TODO: make this nonblocking?
				log.Debugf("WatchFailed: sleeping for %d seconds for %T:%s", int(reconnectTime.Seconds()), source, path)
				time.Sleep(reconnectTime)
				state = WatchInitial
			case WatchInitial:
				log.Debugf("WatchInitial: %T:%s", source, path)
				if stopCh != nil {
					close(stopCh)
				}
				ch = nil

				stopCh = make(chan struct{}, 1)

				ch, err = source.Watch(path, stopCh)
				if err != nil {
					log.Warnf("Failed watching for changes to %T:%s: %s", source, path, err)
					state = WatchFailed
				} else {
					state = Watching
				}
			case Watching:
				log.Debugf("Watching: %T:%s", source, path)

				select {
				case pair := <-ch:
					if pair == nil {
						log.Debug("nil pair returned, restarting watch")
						state = WatchFailed
						// TODO: should this send an empty pair???
					} else {
						// Select on both notify and stop so that if the notify
						// channel is full we don't fail to stop.
						select {
						case retCh <- pair:
						case <-stop:
							return
						}
					}
				case <-stop:
					log.Debug("Stopping watch for %T:%s", source, path)
					stopCh <- struct{}{}
					return
				}

			}
		}
	}()

	return retCh, nil
}

// WatchTreeRobust returns a reliable channel that hides the underlying libkv
// watch because it will fail when files go away
func WatchTreeRobust(source source, path string, stop <-chan struct{}) (<-chan []*store.KVPair, error) {
	retCh := make(chan []*store.KVPair)

	go func() {
		state := WatchInitial
		var stopCh chan struct{}
		var ch <-chan []*store.KVPair
		var err error

		for {
			switch state {
			case WatchFailed:
				// TODO: make this nonblocking?
				log.Printf("WatchFailed: sleeping for %d seconds for %T:%s", int(reconnectTime.Seconds()), source, path)
				time.Sleep(reconnectTime)
				state = WatchInitial
			case WatchInitial:
				log.Printf("WatchInitial: %T:%s", source, path)
				if stopCh != nil {
					close(stopCh)
				}
				ch = nil

				stopCh = make(chan struct{}, 1)

				ch, err = source.WatchTree(path, stopCh)
				if err != nil {
					log.Printf("failed watching for changes to %T:%s: %s", source, path, err)
					state = WatchFailed
				} else {
					state = Watching
				}
			case Watching:
				log.Printf("Watching: %T:%s", source, path)

				select {
				case pairs := <-ch:
					if pairs == nil {
						log.Printf("nil pairs returned, restarting watch")
						state = WatchFailed
						// TODO: should this send an empty pair???
					} else {
						// Select on both notify and stop so that if the notify
						// channel is full we don't fail to stop.
						select {
						case retCh <- pairs:
						case <-stop:
							return
						}
					}
				case <-stop:
					log.Printf("stopping watch for %T:%s", source, path)
					stopCh <- struct{}{}
					return
				}

			}
		}
	}()

	return retCh, nil
}
