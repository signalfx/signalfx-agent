package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"bytes"

	"sync"

	"github.com/docker/libkv/store"
	"github.com/signalfx/neo-agent/config/userconfig"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultInterval is used if not configured
	DefaultInterval = 10
	// DefaultPipeline is used if not configured
	DefaultPipeline = "file"
	// DefaultPollingInterval is the interval in seconds between checking configuration files for changes
	DefaultPollingInterval = 5
	// EnvPrefix is the environment variable prefix
	EnvPrefix = "SFX"

	envMergeConfig = "SFX_MERGE_CONFIG"
	envUserConfig  = "SFX_USER_CONFIG"
)

const (
	// WatchInitial is the initial watch state
	WatchInitial = iota
	// WatchFailed is the watch failed state
	WatchFailed
	// Watching is the normal watching state
	Watching
)

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer   = strings.NewReplacer(".", "_", "-", "_")
	configTimeout = 10 * time.Second
)

// getMergeConfigs returns list of config files to merge from the environment
// variable
func getMergeConfigs() []string {
	var configs []string

	if merge := os.Getenv(envMergeConfig); len(merge) > 1 {
		for _, mergeFile := range strings.Split(merge, ",") {
			configs = append(configs, mergeFile)
		}
	}

	return configs
}

func loadUserConfig(pair *store.KVPair) error {
	var usercon userconfig.UserConfig
	if err := yaml.Unmarshal(pair.Value, &usercon); err != nil {
		return err
	}

	// Check that Kubernetes and Mesosphere are not both configured
	if usercon.Kubernetes != nil && usercon.Mesosphere != nil {
		return errors.New("mesosphere and kubernetes cannot both be set")
	}

	// create kubernetes configuration map
	kubernetes := map[string]interface{}{}

	/// create mesosphere configuration map
	mesosphere := map[string]interface{}{}

	// create cadvisor configuration map
	cadvisor := map[string]interface{}{}

	// create docker-default configuration map
	dockerDefaults := map[string]interface{}{}

	// create staticplugins configuration map
	staticPlugins := map[string]interface{}{}

	// create collectd configuration map
	collectd := map[string]interface{}{
		"staticPlugins": staticPlugins,
	}

	// create plugins configuration map
	plugins := map[string]interface{}{
		"collectd": collectd,
	}

	// create dims plugin configuration map
	dims := map[string]string{}

	// create viper configuration map
	v := map[string]interface{}{
		"plugins":    plugins,
		"dimensions": dims,
	}

	// Parse the and override the default ingest url
	if usercon.IngestURL != "" {
		v["ingesturl"] = usercon.IngestURL
	}

	// parse collectd specific configurations
	if collectdConf := usercon.Collectd; collectdConf != nil {
		usercon.Collectd.Parse(collectd)
	}

	// parse filter configurations
	if filters := usercon.Filter; filters != nil {
		// configure cadvisor filters
		usercon.Filter.Parse(cadvisor)
		// configure docker filters
		usercon.Filter.Parse(dockerDefaults)
		// Since the filters are set let's set docker-default
		staticPlugins["docker-default"] = dockerDefaults
	}

	if kube := usercon.Kubernetes; kube != nil {
		if ok, err := kube.IsValid(); !ok {
			return err
		}
		// parse dimensions from kubernetes configuration
		kube.ParseDimensions(dims)

		// parse kubernetes configurations
		kube.Parse(kubernetes)
		if len(kubernetes) > 0 {
			plugins["kubernetes"] = kubernetes
			//plugins["k8sMonitor"] = kubernetes
		}

		// cadvisor configurations come out of the kubernetes config
		kube.ParseCAdvisor(cadvisor)
		if len(cadvisor) > 0 {
			plugins["cadvisor"] = cadvisor
		}
	}

	// parse proxy configurations
	if proxy := usercon.Proxy; proxy != nil {
		proxyConfig := map[string]string{}
		usercon.Proxy.Parse(proxyConfig)
		if len(proxyConfig) > 0 {
			plugins["proxy"] = proxyConfig
		}
	}

	// Parse mesosphere configuration
	if mesos := usercon.Mesosphere; mesos != nil {
		if ok, err := mesos.IsValid(); !ok {
			return err
		}

		mesos.ParseDimensions(dims)

		// parse mesos configurations
		mesos.Parse(mesosphere)
		if len(mesosphere) > 0 {
			plugins["mesosphere"] = mesosphere
		}
	}

	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)

	if err := viper.MergeConfig(r); err != nil {
		return err
	}
	return nil
}

// load takes a config file in path and performs a reload when the path has changed
func load(path string, reload chan<- struct{}, cb func(pair *store.KVPair) error, mutex *sync.Mutex) error {
	source, path, err := Stores.Get(path)
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		if i != 0 {
			log.Printf("sleeping 5 seconds before retrying EnsureExist")
			time.Sleep(5 * time.Second)
		}
		if err := EnsureExists(source, path); err != nil {
			log.Printf("error for EnsureExist: %s", err)
		} else {
			goto Success
		}
	}

	return fmt.Errorf("failed ensuring %s exists", path)

Success:
	ch, err := ReconnectWatch(source, path, nil)

	if err != nil {
		return err
	}

	select {
	case pair := <-ch:
		log.Printf("config %s loaded", path)
		if err := cb(pair); err != nil {
			return err
		}
	case <-time.After(configTimeout):
		return fmt.Errorf("failed getting initial configuration for %s", path)
	}

	go func() {
		for pair := range ch {
			mutex.Lock()
			if err := cb(pair); err != nil {
				log.Printf("error in callback for %s: %s", path, err)
			}
			mutex.Unlock()
			reload <- struct{}{}
		}
		log.Printf("error: watch stopped for %s", path)
	}()

	return nil

}

// Init loads the config from configfile as well as any merge files from
// environment variable
func Init(configfile string, reload chan<- struct{}, mutex *sync.Mutex) error {
	// Lock so that goroutines kicked off don't modify viper while we're still
	// synchronously loading.
	mutex.Lock()
	defer mutex.Unlock()

	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("pipeline", DefaultPipeline)
	viper.SetDefault("pollingInterval", DefaultPollingInterval)
	viper.SetDefault("ingesturl", "https://ingest.signalfx.com")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(EnvReplacer)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetConfigFile(configfile)

	err := load(configfile, reload, func(pair *store.KVPair) error {
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("error reading agent config: %s", err)
			return err
		}

		return nil
	}, mutex)

	if err != nil {
		return err
	}

	// Configure stores after the base config file has been loaded.
	if err := Stores.Config(viper.Sub("stores")); err != nil {
		return err
	}

	for _, mergeFile := range getMergeConfigs() {
		log.Printf("loading merged config from %s", mergeFile)

		err := load(mergeFile, reload, func(pair *store.KVPair) error {
			log.Printf("%s changed", pair.Key)

			reader := bytes.NewReader(pair.Value)
			if err := viper.MergeConfig(reader); err != nil {
				log.Printf("error merging changes to %s", pair.Key)
				return err
			}

			return nil
		}, mutex)

		if err != nil {
			return err
		}
	}

	// Load user config.
	if userConfigFile := os.Getenv(envUserConfig); userConfigFile != "" {
		log.Printf("loading user configuration from %s", userConfigFile)

		err := load(userConfigFile, reload, func(pair *store.KVPair) error {
			log.Printf("%s changed", userConfigFile)

			if err := loadUserConfig(pair); err != nil {
				log.Printf("failed loading user configuration for %s", userConfigFile)
				return err
			}

			return nil
		}, mutex)

		if err != nil {
			return err
		}
	}

	return nil
}
