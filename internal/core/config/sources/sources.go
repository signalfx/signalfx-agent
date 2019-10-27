// Package sources contains all of the config source logic.  This includes
// logic to get config content from various sources such as the filesystem or a
// KV store.
// It also contains the logic for filling in dynamic values in config.
package sources

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/creasty/defaults"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/consul"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/env"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/etcd2"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/file"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/vault"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/zookeeper"
	"github.com/signalfx/signalfx-agent/internal/core/config/types"
	"github.com/signalfx/signalfx-agent/internal/core/config/validation"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

// SourceConfig represents configuration for various config sources that we
// support.
type SourceConfig struct {
	// Whether to watch config sources for changes.  If this is `true` and any
	// of the config changes (either the main agent.yaml, or remote config
	// values), the agent will dynamically reconfigure itself with minimal
	// disruption.  This is generally better than restarting the agent on
	// config changes since that can result in larger gaps in metric data.  The
	// main disadvantage of watching is slightly greater network and compute
	// resource usage. This option is not itself watched for changes. If you
	// change the value of this option, you must restart the agent.
	Watch bool `yaml:"watch" default:"true"`
	// Configuration for other file sources
	File file.Config `yaml:"file" default:"{}"`
	// Configuration for a Zookeeper remote config source
	Zookeeper *zookeeper.Config `yaml:"zookeeper"`
	// Configuration for an Etcd 2 remote config source
	Etcd2 *etcd2.Config `yaml:"etcd2"`
	// Configuration for a Consul remote config source
	Consul *consul.Config `yaml:"consul"`
	// Configuration for a Hashicorp Vault remote config source
	Vault *vault.Config `yaml:"vault"`
}

// Hash calculates a unique hash value for this config struct
func (sc *SourceConfig) Hash() uint64 {
	hash, err := hashstructure.Hash(sc, nil)
	if err != nil {
		log.WithError(err).Error("Could not get hash of SourceConfig struct")
		return 0
	}
	return hash
}

// SourceInstances returns a map of instantiated sources based on the config
func (sc *SourceConfig) SourceInstances() (map[string]types.ConfigSource, error) {
	sources := make(map[string]types.ConfigSource)

	file := file.New(time.Duration(sc.File.PollRateSeconds) * time.Second)
	sources[file.Name()] = file

	env := env.New()
	sources[env.Name()] = env

	for _, csc := range []types.ConfigSourceConfig{
		sc.Zookeeper,
		sc.Etcd2,
		sc.Consul,
		sc.Vault,
	} {
		if !reflect.ValueOf(csc).IsNil() {
			err := defaults.Set(csc)
			if err != nil {
				panic("Could not set default on source config: " + err.Error())
			}

			if err := validation.ValidateStruct(csc); err != nil {
				return nil, errors.WithMessage(err, "error validating remote config sources")
			}

			if err := csc.Validate(); err != nil {
				return nil, errors.WithMessage(err, fmt.Sprintf("error validating remote config sources"))
			}

			s, err := csc.New()
			if err != nil {
				return nil, errors.WithMessage(err, "error initializing remote config source")
			}
			sources[s.Name()] = s
		}
	}
	return sources, nil
}

func parseSourceConfig(config []byte) (SourceConfig, error) {
	var out struct {
		Sources SourceConfig `yaml:"configSources"`
	}

	err := yaml.Unmarshal(config, &out)
	if err != nil {
		return out.Sources, utils.YAMLErrorWithContext(config, err)
	}

	err = defaults.Set(&out.Sources)
	if err != nil {
		panic("Could not set default on source config: " + err.Error())
	}

	return out.Sources, nil
}

type ConfigFileLoad struct {
	content string
	err     error
}

// ReadConfig reads in the main agent config file and optionally watches for
// changes on it.  It will be returned immediately, along with a channel that
// will be sent any updated config content if watching is enabled.
func StartReadingConfig(ctx context.Context, configPath string) (<-chan []byte, error) {
	// Fetch the config file with a dummy file source since we don't know what
	// poll rate to configure on it yet.
	contentMap, _, err := file.New(1 * time.Second).Get(configPath)
	if err != nil {
		return nil, err
	}
	if len(contentMap) > 1 {
		return nil, fmt.Errorf("path %s resulted in multiple files", configPath)
	}
	if len(contentMap) == 0 {
		return nil, fmt.Errorf("config file %s could not be found", configPath)
	}

	configContent := contentMap[configPath]
	sourceConfig, err := parseSourceConfig(configContent)
	if err != nil {
		return nil, err
	}

	// Now that we know the poll rate for files, we can make a new file source
	// that will be used for the duration of the agent process.
	fileSource := file.New(time.Duration(sourceConfig.File.PollRateSeconds) * time.Second)
	fileSourceIter := NewConfigSourceIterator(fileSource, configPath)

	changes := make(chan []byte)

	go func() {
		defer close(changes)
		for {
			contentMap, err := fileSourceIter.Next(ctx)

			if ctx.Err() != nil {
				return
			}

			if err != nil {
				log.WithError(err).Error("Could not read config file")
				time.Sleep(5 * time.Second)
			} else {
				changes <- contentMap[configPath]
			}

			if !sourceConfig.Watch {
				return
			}

			log.Info("Watching for config file changes")
		}
	}()

	return changes, nil
}

// DynamicValueProvider handles setting up and providing dynamic values from
// remote config sources.
type DynamicValueProvider struct {
	lastRemoteConfigSourceHash uint64
	sources                    map[string]types.ConfigSource
	changes                    chan []byte
	ctx                        context.Context
	cancel                     context.CancelFunc
}

func NewDynamicValueProvider() *DynamicValueProvider {
	return &DynamicValueProvider{
		changes: make(chan []byte),
	}
}

func (dvp *DynamicValueProvider) Changes() <-chan []byte {
	return dvp.changes
}

// ReadDynamicValues takes the config file content and processes it for any
// dynamic values of the form `{"#from": ...`.  It returns a YAML document that
// contains the rendered values.  It will optionally watch the sources of any
// dynamic values configured and send updated YAML docs on the returned
// channel.
func (dvp *DynamicValueProvider) ReadDynamicValues(ctx context.Context, configContent []byte) error {
	// Shut down any previous reads
	if dvp.cancel != nil {
		dvp.cancel()
	}
	dvp.ctx, dvp.cancel = context.WithCancel(ctx)

	sourceConfig, err := parseSourceConfig(configContent)
	if err != nil {
		return err
	}

	hash := sourceConfig.Hash()
	if hash != dvp.lastRemoteConfigSourceHash {
		for name, source := range dvp.sources {
			if stoppable, ok := source.(types.Stoppable); ok {
				log.Infof("Stopping stale %s remote config source", name)
				if err := stoppable.Stop(); err != nil {
					log.WithError(err).Errorf("Could not stop stale %s remote config source", name)
				}
			}
		}
		dvp.sources, err = sourceConfig.SourceInstances()
		if err != nil {
			return err
		}
		dvp.lastRemoteConfigSourceHash = hash
	}

	// This is what the cacher will notify on with the names of dynamic value
	// paths that change
	pathChanges := make(chan string)

	cachers := make(map[string]*configSourceCacher)
	for name, source := range dvp.sources {
		cacher := newConfigSourceCacher(dvp.ctx, source, pathChanges, sourceConfig.Watch)
		cachers[name] = cacher
	}

	resolver := newResolver(cachers)

	go func() {
		for {
			renderedContent, err := renderDynamicValues(configContent, resolver.Resolve)
			if err != nil {
				log.WithError(err).Error("Could not render dynamic values in config after change")
				time.Sleep(5 * time.Second)
				continue
			}

			dvp.changes <- renderedContent
			if !sourceConfig.Watch {
				return
			}

			select {
			case path := <-pathChanges:
				log.Debugf("Dynamic value path %s changed", path)
				continue
			case <-dvp.ctx.Done():
				return
			}
		}
	}()

	return nil
}
