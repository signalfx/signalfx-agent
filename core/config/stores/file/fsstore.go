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

type FSStore struct {
	watcher *watcher.PollingWatcher
}

func New() *FSStore {
	// TODO: configurable polling interval
	f := &FSStore{watcher.NewPollingWatcher(5 * time.Second)}
	f.watcher.Start()
	return f
}

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

func (f *FSStore) Put(key string, value []byte, opts *store.WriteOptions) error {
	return ioutil.WriteFile(key, value, 0644)
}

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

func (f *FSStore) Get(path string) (*store.KVPair, error) {
	log.Debugf("FSStore.Get(%s)", path)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &store.KVPair{Key: path, Value: data, LastIndex: 0}, nil
}

func (f *FSStore) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return os.IsExist(err), nil
}

func (f *FSStore) Watch(path string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
	return f.watcher.Watch(path, stopCh)
}

func (f *FSStore) WatchTree(path string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	return f.watcher.WatchTree(path, stopCh)
}

func (f *FSStore) Close() {
	f.watcher.Close()
}
