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
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultInterval is used if not configured
	DefaultInterval = 10
	// DefaultPipeline is used if not configured
	DefaultPipeline = "docker"
	// DefaultPollingInterval is the interval in seconds between checking configuration files for changes
	DefaultPollingInterval = 5
	// EnvPrefix is the environment variable prefix
	EnvPrefix = "SFX"

	envMergeConfig = "SFX_MERGE_CONFIG"
	envUserConfig  = "SFX_USER_CONFIG"
)

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer   = strings.NewReplacer(".", "_", "-", "_")
	configTimeout = 10 * time.Second
)

type userConfig struct {
	Proxy *struct {
		HTTP  string
		HTTPS string
		Skip  string
	}
	Kubernetes *struct {
		IgnoreTLSVerify bool `yaml:"ignoreTLSVerify,omitempty"`
		Role            string
		Cluster         string
	}
}

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

	source, path, err := Stores.Get(configfile)
	if err != nil {
		return err
	}

	ch, err := source.Watch(path, nil)
	if err != nil {
		return err
	}

	select {
	case <-ch:
		log.Printf("loading agent config %s", configfile)

		if err := viper.ReadInConfig(); err != nil {
			log.Printf("error reading agent config: %s", err)
			return err
		}
	case <-time.After(configTimeout):
		return fmt.Errorf("failed getting initial agent configuration %s", configfile)
	}

	// Configure stores after the base config file has been loaded.
	if err := Stores.Config(viper.Sub("stores")); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ch:
				mutex.Lock()
				log.Printf("reloading agent config %s", configfile)

				if err := viper.ReadInConfig(); err != nil {
					log.Printf("error reading agent config: %s", err)
				} else {
					reload <- struct{}{}
				}
				mutex.Unlock()
			}
		}
	}()

	if err := Stores.Config(viper.Sub("stores")); err != nil {
		return err
	}

	for _, mergeFile := range getMergeConfigs() {
		log.Printf("loading merged config from %s", mergeFile)

		source, path, err := Stores.Get(mergeFile)
		if err != nil {
			return err
		}

		ch, err := source.Watch(path, nil)
		if err != nil {
			return err
		}

		select {
		case pair := <-ch:
			reader := bytes.NewReader(pair.Value)
			if err := viper.MergeConfig(reader); err != nil {
				log.Printf("error merging changes to %s", pair.Key)
				return err
			}
		case <-time.After(configTimeout):
			return fmt.Errorf("failed getting initial configuration for %s", mergeFile)
		}

		go func() {
			for {
				select {
				case pair := <-ch:
					mutex.Lock()
					log.Printf("%s changed", pair.Key)

					reader := bytes.NewReader(pair.Value)
					if err := viper.MergeConfig(reader); err != nil {
						log.Printf("error merging changes to %s", pair.Key)
					} else {
						reload <- struct{}{}
					}
					mutex.Unlock()
				}
			}
		}()
	}

	// Load user config.
	if userConfigFile := os.Getenv(envUserConfig); userConfigFile != "" {
		log.Printf("loading user configuration from %s", userConfigFile)

		source, path, err := Stores.Get(userConfigFile)
		if err != nil {
			return err
		}

		ch, err := source.Watch(path, nil)
		if err != nil {
			return err
		}

		loadConfig := func(pair *store.KVPair) error {
			var usercon userConfig
			if err := yaml.Unmarshal(pair.Value, &usercon); err != nil {
				return err
			}

			plugins := map[string]interface{}{}

			v := map[string]interface{}{
				"plugins": plugins,
			}

			if kube := usercon.Kubernetes; kube != nil {
				if kube.Cluster == "" {
					return errors.New("kubernetes.cluster missing")
				}
				if kube.Role != "worker" && kube.Role != "master" {
					return errors.New("kubernetes.role must be worker or master")
				}
				dims := map[string]string{}
				kubernetes := map[string]interface{}{}

				dims["kubernetes_cluster"] = kube.Cluster
				dims["kubernetes_role"] = kube.Role
				kubernetes["ignoretlsverify"] = kube.IgnoreTLSVerify
				plugins["kubernetes"] = kubernetes
				v["dimensions"] = dims
			}

			if proxy := usercon.Proxy; proxy != nil {
				proxyConfig := map[string]string{}
				proxyConfig["plugins.proxy.http"] = proxy.HTTP
				proxyConfig["plugins.proxy.https"] = proxy.HTTPS
				proxyConfig["plugins.proxy.skip"] = proxy.Skip
				plugins["proxy"] = proxyConfig
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

		select {
		case pair := <-ch:
			if err := loadConfig(pair); err != nil {
				return err
			}
		case <-time.After(configTimeout):
			return fmt.Errorf("failed getting initial configuration for %s", userConfigFile)
		}

		go func() {
			for {
				select {
				case pair := <-ch:
					mutex.Lock()
					log.Printf("%s changed", userConfigFile)

					if err := loadConfig(pair); err != nil {
						log.Printf("failed loading configuration for %s", userConfigFile)
					} else {
						reload <- struct{}{}
					}
					mutex.Unlock()
				}
			}
		}()
	}

	return nil
}
