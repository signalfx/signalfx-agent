// Package file contains a store that uses the local filesystem
package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/docker/libkv/store"
	"github.com/signalfx/neo-agent/core/config/stores/file/watcher"
)

// FSStore is an libkv store implementation for filesystem paths
type FSStore struct {
	watcher *watcher.PollingWatcher
}

// New creates a new FSStore
func New() *FSStore {
	// TODO: configurable polling interval
	f := &FSStore{watcher.NewPollingWatcher(5 * time.Second)}
	f.watcher.Start()
	return f
}

// List returns all of the files in a directory
func (f *FSStore) List(dir string) ([]*store.KVPair, error) {
	files, err := ioutil.ReadDir(dir)
	var ret []*store.KVPair

	if err != nil {
		return nil, err
	}

	for _, f := range files {
		fullPath := path.Join(dir, f.Name())
		bytes, err := ioutil.ReadFile(fullPath)
		if err != nil {
			continue
		}
		ret = append(ret, &store.KVPair{Key: fullPath, Value: bytes, LastIndex: 0})
	}

	return ret, nil
}

// Put writes a file, but not necessarily atomic
func (f *FSStore) Put(key string, value []byte, opts *store.WriteOptions) error {
	return ioutil.WriteFile(key, value, 0644)
}

// AtomicPut writes a file atomically
func (f *FSStore) AtomicPut(key string, value []byte, previous *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error) {
	if previous != nil {
		return false, nil, fmt.Errorf("only empty atomic put is supported for filesystems")
	}
	file, err := os.OpenFile(key, os.O_CREATE, 0644)
	if err != nil {
		return false, nil, err
	}

	file.Close()

	return true, &store.KVPair{Key: key, Value: value, LastIndex: 0}, nil
}

// Get synchronously gets a path
func (f *FSStore) Get(path string) (*store.KVPair, error) {
	log.Debugf("FSStore.Get(%s)", path)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &store.KVPair{Key: path, Value: data, LastIndex: 0}, nil
}

// Exists tests if a path points to existing content
func (f *FSStore) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return os.IsExist(err), nil
}

// Watch a single file
func (f *FSStore) Watch(path string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
	return f.watcher.Watch(path, stopCh)
}

// WatchTree watches a directory recursively
func (f *FSStore) WatchTree(path string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	return f.watcher.WatchTree(path, stopCh)
}

// Close closes the store and stops watches
func (f *FSStore) Close() {
	f.watcher.Close()
}
