// Package stores holds logic for generic access to various configuration
// sources (e.g. filesystem, zookeeper, etc.).
package stores

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	"github.com/docker/libkv/store/zookeeper"
	"github.com/mitchellh/mapstructure"
	"github.com/signalfx/neo-agent/core/config/stores/file"
	log "github.com/sirupsen/logrus"
)

// MetaStore is a higher-order store that delegates to the appropriate store implementation
type MetaStore struct {
	stores map[string]Source
}

// NewMetaStore creates a new metastore instance with a default filesystem source
func NewMetaStore() *MetaStore {
	store := &MetaStore{map[string]Source{
		"file": file.New(),
	}}

	return store
}

// ConfigureFromEnv configures the metastore from envvars since the config file
// might be loaded from a source (e.g. zookeeper) that requires configuration
// before it can be read.  It solves the chicken/egg problem of getting
// configuration from a source that must be itself configured first.
func (ms *MetaStore) ConfigureFromEnv() {
	conf := make(map[string]map[string]interface{})
	if hosts := os.Getenv("SFX_STORES_ZK_HOSTS"); hosts != "" {
		conf["zookeeper"] = map[string]interface{}{}
		conf["zookeeper"]["hosts"] = hosts
	}
	ms.Configure(conf)
}

// Configure a metastore and the associated sub-stores.
func (ms *MetaStore) Configure(conf map[string]map[string]interface{}) error {
	for name, storeConf := range conf {
		switch name {
		case "filesystem":
			log.Debug("Configuring filesystem store")

			var fs struct {
				Interval float32
			}
			if err := mapstructure.Decode(storeConf, &fs); err != nil {
				return err
			}
			// TODO: reconfigure if already present
			if _, ok := ms.stores[name]; !ok {
				ms.stores[name] = file.New()
			}
		case "zookeeper":
			log.Debug("Configuring zookeeper store")
			var zk struct {
				// Hosts may be an array or a comma separated list of hosts.
				Hosts interface{}
			}
			if err := mapstructure.Decode(storeConf, &zk); err != nil {
				return err
			}
			if _, ok := ms.stores[name]; !ok {
				// TODO: reconfigure
				log.Infof("Added new store for %s", name)
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
				ms.stores[name] = store
			}

		default:
			return fmt.Errorf("unknown type '%s'", name)
		}
	}

	return nil
}

// GetSourceAndPath returns a source instance and the parsed out path if it's
// valid and the source exists, otherwise err is set
func (ms *MetaStore) GetSourceAndPath(uri string) (source Source, path string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	scheme := u.Scheme
	if scheme == "" {
		scheme = filesystemScheme
	}

	if s, ok := ms.stores[scheme]; ok {
		if u.Host != "" {
			return s, u.Host + u.Path, nil
		}

		return s, u.Path, nil
	}

	return nil, "", fmt.Errorf("unknown source '%s'", scheme)
}

// Close closes all the underlying sources
func (ms *MetaStore) Close() {
	for _, source := range ms.stores {
		source.Close()
	}
}

// WatchPath will watch the provided uri (or a plain filesystem path) and send
// both the initial load and updates to the returned channel.  The key of the
// KVPair will be the filename and the Value will be the file contents.
func (ms *MetaStore) WatchPath(uri string) (<-chan *store.KVPair, func(), error) {
	source, path, err := ms.GetSourceAndPath(uri)
	if err != nil {
		return nil, nil, err
	}

	// TODO: figure out exactly why this commented code was necessary and
	// perhaps find another way to mitigate it
	/*tries := 0
	for {
		if err := EnsureExists(source, path); err != nil {
			tries++

			if tries > 3 {
				return nil, fmt.Errorf("failed ensuring %s exists", path)
			}

			log.WithFields(log.Fields{
				"error": err,
				"path":  path,
			}).Error("Error loading config file")
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}*/

	stopCh := make(chan struct{})
	stopCalled := false

	ch, err := WatchRobust(source, path, stopCh)
	if err != nil {
		return nil, nil, err
	}

	stop := func() {
		if !stopCalled {
			stopCalled = true
			stopCh <- struct{}{}
		}
	}

	return ch, stop, nil
}
