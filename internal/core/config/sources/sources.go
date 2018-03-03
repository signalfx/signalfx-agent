// Package sources contains all of the config source logic.  This includes
// logic to get config content from various sources such as the filesystem or a
// KV store.
// It also contains the logic for filling in dynamic values in config.
package sources

import (
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/consul"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/etcd2"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/file"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/types"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/zookeeper"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

// SourceConfig represents configuration for various config sources that we
// support.
type SourceConfig struct {
	// Whether to watch config sources for changes.  If this is `true` and any
	// of the config changes, the agent will dynamically reconfigure itself
	// with minimal disruption.  This is generally better than restarting the
	// agent on config changes since that can result in larger gaps in metric
	// data.  The main disadvantage of watching is slightly greater network and
	// compute resource usage.
	// This option itself ironically enough is not subject to watching and
	// changing it to false after the agent was started with it true will
	// require an agent restart.
	Watch bool `yaml:"watch" default:"true"`
	// Configuration for other file sources
	File file.Config `yaml:"file" default:"{}"`
	// Configuration for a Zookeeper remote config source
	Zookeeper *zookeeper.Config `yaml:"zookeeper"`
	// Configuration for an Etcd 2 remote config source
	Etcd2 *etcd2.Config `yaml:"etcd2"`
	// Configuration for a Consul remote config source
	Consul *consul.Config `yaml:"consul"`
}

// SourceInstances returns a map of instantiated sources based on the config
func (sc *SourceConfig) SourceInstances() (map[string]types.ConfigSource, error) {
	sources := make(map[string]types.ConfigSource)

	file := file.New(time.Duration(sc.File.PollRateSeconds) * time.Second)
	sources[file.Name()] = file

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

func parseSourceConfig(config []byte) (SourceConfig, error) {
	var out struct {
		Sources SourceConfig `yaml:"configSources"`
	}
	err := defaults.Set(&out.Sources)
	if err != nil {
		panic("Could not set default on source config: " + err.Error())
	}
	err = yaml.Unmarshal(config, &out)
	if err != nil {
		return out.Sources, err
	}

	return out.Sources, nil
}

// ReadConfig reads in the main agent config file and optionally watches for
// changes on it.  It will be returned immediately, along with a channel that
// will be sent any updated config content if watching is enabled.
func ReadConfig(configPath string, stop <-chan struct{}) ([]byte, <-chan []byte, error) {
	// Fetch the config file with a dummy file source since we don't know what
	// poll rate to configure on it yet.
	contentMap, version, err := file.New(1 * time.Second).Get(configPath)
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
	sourceConfig, err := parseSourceConfig(configContent)
	if err != nil {
		return nil, nil, err
	}

	// Now that we know the poll rate for files, we can make a new file source
	// that will be used for the duration of the agent process.
	fileSource := file.New(time.Duration(sourceConfig.File.PollRateSeconds) * time.Second)

	if sourceConfig.Watch {
		log.Info("Watching for config file changes")
		changes := make(chan []byte)
		go func() {
			for {
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
		return configContent, changes, nil
	}

	return configContent, nil, nil
}

// ReadDynamicValues takes the config file content and processes it for any
// dynamic values of the form `{"#from": ...`.  It returns a YAML document that
// contains the rendered values.  It will optionally watch the sources of any
// dynamic values configured and send updated YAML docs on the returned
// channel.
func ReadDynamicValues(configContent []byte, stop <-chan struct{}) ([]byte, <-chan []byte, error) {
	sourceConfig, err := parseSourceConfig(configContent)
	if err != nil {
		return nil, nil, err
	}

	sources, err := sourceConfig.SourceInstances()
	if err != nil {
		return nil, nil, err
	}

	// This is what the cacher will notify on with the names of dynamic value
	// paths that change
	pathChanges := make(chan string)

	cachers := make(map[string]*configSourceCacher)
	for name, source := range sources {
		cacher := newConfigSourceCacher(source, pathChanges, stop, sourceConfig.Watch)
		cachers[name] = cacher
	}

	resolver := newResolver(cachers)

	renderedContent, err := renderDynamicValues(configContent, resolver.Resolve)
	if err != nil {
		return nil, nil, err
	}

	var changes chan []byte
	if sourceConfig.Watch {
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
