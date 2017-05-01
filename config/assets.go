package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"

	"os"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/zookeeper"
	"github.com/signalfx/neo-agent/watchers"
	"github.com/spf13/viper"
)

// Stores is a global metastore instance
var Stores = newStores()

const defaultScheme = "fs"

// reconnectTime is time between store reconnect attempts
var reconnectTime = 30 * time.Second

type source interface {
	Get(string) (*store.KVPair, error)
	Watch(string, <-chan struct{}) (<-chan *store.KVPair, error)
	WatchTree(string, <-chan struct{}) (<-chan []*store.KVPair, error)
	Exists(string) (bool, error)
	Put(key string, value []byte, opts *store.WriteOptions) error
	AtomicPut(key string, value []byte, previous *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error)
	Close()
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

type fs struct {
	watcher *watchers.PollingWatcher
}

func newFs() *fs {
	// TODO: configurable polling interval
	f := &fs{watchers.NewPollingWatcher(5 * time.Second)}
	f.watcher.Start()
	return f
}

func (f *fs) Put(key string, value []byte, opts *store.WriteOptions) error {
	return ioutil.WriteFile(key, value, 0644)
}

func (f *fs) AtomicPut(key string, value []byte, previous *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error) {
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

func (f *fs) Get(path string) (*store.KVPair, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &store.KVPair{Key: path, Value: data, LastIndex: 0}, nil
}

func (f *fs) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return os.IsExist(err), nil
}

func (f *fs) Watch(path string, stopCh <-chan struct{}) (<-chan *store.KVPair, error) {
	return f.watcher.Watch(path, stopCh)
}

func (f *fs) WatchTree(path string, stopCh <-chan struct{}) (<-chan []*store.KVPair, error) {
	return f.watcher.WatchTree(path, stopCh)
}

func (f *fs) Close() {
	f.watcher.Close()
}

// reconnectWatch returns a reliable channel that hides the underlying libkv
// watch because it will fail when files go away
func reconnectWatch(source source, path string, stop <-chan struct{}) (<-chan *store.KVPair, error) {
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

				ch, err = source.Watch(path, stopCh)
				if err != nil {
					log.Printf("failed watching for changes to %T:%s: %s", source, path, err)
					state = WatchFailed
				} else {
					state = Watching
				}
			case Watching:
				log.Printf("Watching: %T:%s", source, path)

				select {
				case pair := <-ch:
					if pair == nil {
						log.Printf("nil pair returned, restarting watch")
						state = WatchFailed
						// TODO: should this send an empty pair???
					} else {
						retCh <- pair
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

type metaStore struct {
	stores map[string]source
}

// newStores creates a new metastore instance with a default filesystem source
func newStores() *metaStore {
	store := &metaStore{map[string]source{
		"fs": newFs(),
	}}

	// Kind of a hack. Unfortunately UnmarshalKey used in the configuration
	// doesn't support environment variable overrides. This can't be set in an
	// merge file because merge files might contain a zk:// merge and this has
	// to be configured before that.
	if hosts := os.Getenv("SFX_STORES_ZK_HOSTS"); hosts != "" {
		viper.Set("stores.zk.type", "zookeeper")
		viper.Set("stores.zk.hosts", hosts)
	}

	return store
}

// Config configures metastore from a viper config
func (s *metaStore) Config(config *viper.Viper) error {
	// TODO: handle reload (have to shut down unused ones or reconfigure fs)
	if config == nil {
		return errors.New("stores configuration is nil")
	}
	var sources map[string]struct{}
	if err := config.Unmarshal(&sources); err != nil {
		return err
	}

	for source := range sources {
		typ := config.GetString(source + ".type")
		switch typ {
		case "filesystem":
			log.Print("configuring filesystem store")
			var fs struct {
				Interval float32
			}
			if err := config.UnmarshalKey(source, &fs); err != nil {
				return err
			}
			// TODO: reconfigure if already present
			if _, ok := s.stores[source]; !ok {
				s.stores[source] = newFs()
			}
		case "zookeeper":
			log.Print("configuring zookeeper store")
			var zk struct {
				// Hosts may be an array or a comma separated list of hosts.
				Hosts interface{}
			}
			if err := config.UnmarshalKey(source, &zk); err != nil {
				return err
			}
			if _, ok := s.stores[source]; !ok {
				// TODO: reconfigure
				log.Printf("added new store for %s %s", source, typ)
				config := store.Config{}
				zookeeper.Register()
				var zkHosts []string

				if zk.Hosts == nil {
					return errors.New("zookeeper hosts cannot be empty")
				}

				switch hosts := zk.Hosts.(type) {
				case string:
					zkHosts = strings.Split(hosts, ",")
				case []interface{}:
					for _, s := range hosts {
						if str, ok := s.(string); ok {
							zkHosts = append(zkHosts, str)
						}
					}
				default:
					return errors.New("unknown hosts type")
				}

				store, err := libkv.NewStore(store.ZK, zkHosts, &config)
				if err != nil {
					return err
				}
				s.stores[source] = store
			}

		default:
			return fmt.Errorf("unknown type '%s'", source)
		}
	}

	return nil
}

// Get returns a source instance and the parsed out path if it's valid and
// the source exists, otherwise err is set
func (s *metaStore) Get(uri string) (source source, path string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	scheme := u.Scheme

	if scheme == "" {
		scheme = defaultScheme
	}

	if s, ok := s.stores[scheme]; ok {
		if u.Host != "" {
			return s, u.Host + u.Path, nil
		}

		return s, u.Path, nil
	}

	return nil, "", fmt.Errorf("unknown source '%s'", scheme)
}

// Close closes all the underlying sources
func (s *metaStore) Close() {
	for _, source := range s.stores {
		source.Close()
	}
}
