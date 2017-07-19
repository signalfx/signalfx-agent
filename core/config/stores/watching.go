package stores

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"

	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/zookeeper"
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

// AssetSpec specifies assets to sync
type AssetSpec struct {
	Files map[string]string
	Dirs  map[string]string
}

// NewAssetSpec creates a new AssetSpec
func NewAssetSpec() *AssetSpec {
	return &AssetSpec{map[string]string{}, map[string]string{}}
}

// AssetSyncer maintains a set of assets from any number of different sources
// and keeps them up-to-date. Notifications can be sent when any of them change.
// Notification includes all of the assets in the syncer, not just the changed
// ones.
type AssetSyncer struct {
	mutex   sync.Mutex
	wg      sync.WaitGroup
	changed chan changeNotification
	dirs    map[string]*dirTracker
	stop    chan struct{}
	store   *MetaStore
}

// NewAssetSyncer creates a new instance
func NewAssetSyncer(store *MetaStore) *AssetSyncer {
	return &AssetSyncer{
		stop:    make(chan struct{}),
		dirs:    map[string]*dirTracker{},
		changed: make(chan changeNotification),
		store:   store,
	}
}

// AssetsView is a read-only view
type AssetsView struct {
	Dirs map[string][]*store.KVPair
}
type assetTracker struct {
	uri  string
	stop chan struct{}
}

type fileTracker struct {
	assetTracker
	files *store.KVPair
}

type dirTracker struct {
	assetTracker
	dirs         []*store.KVPair
	reloadChStop chan struct{}
}

type changeNotification struct {
	pairs []*store.KVPair
	name  string
}

func (a *AssetSyncer) add(name, dir string) error {
	// added
	stopWatcher := make(chan struct{})
	stopAsset := make(chan struct{})
	var reloadCh <-chan *store.KVPair
	var stopReload chan struct{}

	source, path, err := a.store.GetSourceAndPath(dir)
	if err != nil {
		return err
	}
	ch, err := WatchTreeRobust(source, path, stopWatcher)
	if err != nil {
		return err
	}

	// Only send reload notification for ZooKeeper since other backends don't
	// need it.
	if _, ok := source.(*zookeeper.Zookeeper); ok {
		reloadPath := path + "/.reload"
		log.Debugf("Watching for reloads at %s", reloadPath)

		if err := EnsureExists(source, path); err != nil {
			stopWatcher <- struct{}{}
			return err
		}

		stopReload = make(chan struct{})

		reloadCh, err = WatchRobust(source, reloadPath, stopReload)

		if err != nil {
			stopWatcher <- struct{}{}
			return err
		}
	}

	a.dirs[name] = &dirTracker{assetTracker{dir, stopAsset}, nil, stopReload}

	go func() {
		if stopReload != nil {
			defer close(stopReload)
		}
		defer close(stopWatcher)
		defer close(stopAsset)

		for {
			select {
			case pair := <-reloadCh:
				if pair == nil {
					log.Printf("unexpected reload channel returned nil")
					continue
				}

				pairs, err := source.List(path)
				if err != nil {
					log.Printf("unable to list from reload: %s", err)
					continue
				}
				select {
				// Select on both changed and stop so that if the notify channel
				// is full we don't fail to stop.
				case a.changed <- changeNotification{name: name, pairs: pairs}:
				case <-stopAsset:
					return
				}
			case pairs := <-ch:
				if pairs == nil {
					log.Printf("error: unexpected stopping asset sync for %s", dir)
					return
				}
				select {
				// Select on both changed and stop so that if the notify channel
				// is full we don't fail to stop.
				case a.changed <- changeNotification{name: name, pairs: pairs}:
				case <-stopAsset:
					return
				}
			case <-stopAsset:
				log.Printf("stopping asset sync for %s", name)
				stopWatcher <- struct{}{}
				return
			}
		}
	}()

	return nil
}

func (tracker *dirTracker) stopWatch() {
	tracker.stop <- struct{}{}
	if tracker.reloadChStop != nil {
		tracker.reloadChStop <- struct{}{}
	}
}

// Update assets to watch
func (a *AssetSyncer) Update(w *AssetSpec) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if w == nil {
		for name, tracker := range a.dirs {
			tracker.stopWatch()
			delete(a.dirs, name)
		}
		return nil
	}
	for name, dir := range w.Dirs {
		if tracker, ok := a.dirs[dir]; ok {
			if dir != tracker.uri {
				// changed
				log.Printf("changing %s from %s to %s", name, tracker.uri, dir)
				tracker.stopWatch()
				if err := a.add(name, dir); err != nil {
					return err
				}
			}
		} else {
			// added
			if err := a.add(name, dir); err != nil {
				return err
			}
		}
	}

	for dir, tracker := range a.dirs {
		if _, ok := w.Dirs[dir]; !ok {
			// removed
			tracker.stopWatch()
			delete(a.dirs, dir)
		}
	}

	return nil
}

// Start the sync
func (a *AssetSyncer) Start(cb func(view *AssetsView)) {
	a.wg.Add(1)

	go func() {
		for {
			select {
			case notif := <-a.changed:
				func() {
					a.mutex.Lock()

					tracker, ok := a.dirs[notif.name]
					if !ok {
						log.Printf("missing entry for %s", notif.name)
						a.mutex.Unlock()
						return
					}

					tracker.dirs = notif.pairs

					view := &AssetsView{Dirs: map[string][]*store.KVPair{}}
					for key, value := range a.dirs {
						view.Dirs[key] = value.dirs
					}
					a.mutex.Unlock()
					cb(view)
				}()
			case <-a.stop:
				a.Update(nil)
				a.wg.Done()
				return
			}
		}
	}()
}

// Stop the sync
func (a *AssetSyncer) Stop() {
	a.stop <- struct{}{}
	a.wg.Wait()
}

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
