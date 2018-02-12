// Package sources contains all of the config source logic.  This includes
// logic to get config content from various sources such as the filesystem or a
// KV store.
// It also contains the logic for filling in dynamic values in config.
package sources

import (
	"fmt"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config/sources/consul"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/etcd2"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/types"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/zookeeper"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

// SourceConfig represents configuration for various config sources that we
// support.
type sourceConfig struct {
	Zookeeper *zookeeper.Config `yaml:"zookeeper"`
	Etcd2     *etcd2.Config     `yaml:"etcd2"`
	//Etcd3 etcd3.Config `yaml:"etcd3"`
	Consul *consul.Config `yaml:"consul"`
}

// Sources returns a map of instantiated sources based on the config
func (sc *sourceConfig) Sources() (map[string]types.ConfigSource, error) {
	sources := make(map[string]types.ConfigSource)
	if sc.Zookeeper != nil {
		zk := zookeeper.New(sc.Zookeeper)
		sources[zk.Name()] = zk
	}
	if sc.Etcd2 != nil {
		e2, err := etcd2.New(sc.Etcd2)
		if err != nil {
			return nil, err
		}
		sources[e2.Name()] = e2
	}
	if sc.Consul != nil {
		c, err := consul.New(sc.Consul)
		if err != nil {
			return nil, err
		}
		sources[c.Name()] = c
	}
	return sources, nil
}

func parseSourceConfig(config []byte) (sourceConfig, error) {
	var out struct {
		Sources sourceConfig `yaml:"configSources"`
	}
	err := yaml.Unmarshal(config, &out)
	return out.Sources, err
}

// ReadConfig reads in the main agent config file and optionally watches for
// changes on it.  It will be returned immediately, along with a channel that
// will be sent any updated config content if watching is enabled.
func ReadConfig(configPath string, fileSource types.ConfigSource,
	stop <-chan struct{}, shouldWatch bool) ([]byte, <-chan []byte, error) {

	contentMap, version, err := fileSource.Get(configPath)
	if err != nil {
		return nil, nil, err
	}
	if len(contentMap) > 1 {
		return nil, nil, fmt.Errorf("Path %s resulted in multiple files", configPath)
	}
	if len(contentMap) == 0 {
		return nil, nil, fmt.Errorf("Config file %s could not be found", configPath)
	}

	configContent := contentMap[configPath]

	var changes chan []byte
	if shouldWatch {
		changes = make(chan []byte)
		go func() {
			for {
				log.Debug("Waiting for config file to change")
				err := fileSource.WaitForChange(configPath, version, stop)
				log.Info("Config file changed")

				if utils.IsSignalChanClosed(stop) {
					return
				}
				if err != nil {
					log.WithError(err).Error("Could not wait for changes to config file")
					time.Sleep(5 * time.Second)
					continue
				}

				contentMap, version, err = fileSource.Get(configPath)
				if err != nil {
					log.WithError(err).Error("Could not get config file after it was changed")
					time.Sleep(5 * time.Second)
					continue
				}

				changes <- contentMap[configPath]
			}
		}()
	}

	return configContent, changes, nil
}

// ReadDynamicValues takes the config file content and processes it for any
// dynamic values of the form `{"#from": ...`.  It returns a YAML document that
// contains the rendered values.  It will optionally watch the sources of any
// dynamic values configured and send updated YAML docs on the returned
// channel.
func ReadDynamicValues(configContent []byte, fileSource types.ConfigSource,
	stop <-chan struct{}, shouldWatch bool) ([]byte, <-chan []byte, error) {

	sources := map[string]types.ConfigSource{
		fileSource.Name(): fileSource,
	}

	sourceConfig, err := parseSourceConfig(configContent)
	if err != nil {
		return nil, nil, err
	}

	configuredSources, err := sourceConfig.Sources()
	if err != nil {
		return nil, nil, err
	}

	for name, source := range configuredSources {
		sources[name] = source
	}

	// This is what the cacher will notify on with the names of dynamic value
	// paths that change
	pathChanges := make(chan string)

	cachers := make(map[string]*configSourceCacher)
	for name, source := range sources {
		cacher := newConfigSourceCacher(source, pathChanges, stop, shouldWatch)
		cachers[name] = cacher
	}

	resolver := newResolver(cachers)

	renderedContent, err := renderDynamicValues(configContent, resolver.Resolve)
	if err != nil {
		return nil, nil, err
	}

	var changes chan []byte
	if shouldWatch {
		changes = make(chan []byte)

		go func() {
			for {
				select {
				case path := <-pathChanges:
					log.Debugf("Dynamic value path %s changed", path)

					renderedContent, err = renderDynamicValues(configContent, resolver.Resolve)
					if err != nil {
						log.WithError(err).Error("Could not render dynamic values in config after change")
						time.Sleep(5 * time.Second)
						continue
					}

					changes <- renderedContent
				case <-stop:
					return
				}
			}
		}()
	}

	return renderedContent, changes, nil
}
