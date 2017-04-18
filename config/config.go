package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"io/ioutil"

	"bytes"

	"github.com/signalfx/neo-agent/watchers"
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
	EnvReplacer = strings.NewReplacer(".", "_", "-", "_")
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

// WatchForChanges watches for changes to configuration files and reloads on change
func WatchForChanges(watcher *watchers.PollingWatcher, configfile string) {
	// Watch base config and merged config for changes. If either changes reload
	// viper config.
	configFiles := append(getMergeConfigs(), configfile)
	if userConfig := os.Getenv(envUserConfig); userConfig != "" {
		configFiles = append(configFiles, userConfig)
	}
	log.Printf("watching for changes to %v", configFiles)

	watcher.Watch(nil, configFiles)
	watcher.Start()
}

// getMergeConfigs returns list of config files to merge from the environment
// variable
func getMergeConfigs() []string {
	var configs []string

	if merge := os.Getenv(envMergeConfig); len(merge) > 1 {
		for _, mergeFile := range strings.Split(merge, ":") {
			configs = append(configs, mergeFile)
		}
	}

	return configs
}

// Load loads the config from configfile as well as any merge files from
// environment variable
func Load(configfile string) error {
	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("pipeline", DefaultPipeline)
	viper.SetDefault("pollingInterval", DefaultPollingInterval)
	viper.SetDefault("ingesturl", "https://ingest.signalfx.com")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(EnvReplacer)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetConfigFile(configfile)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	for _, mergeFile := range getMergeConfigs() {
		if file, err := os.Open(mergeFile); err == nil {
			defer file.Close()
			log.Printf("merging config %s", mergeFile)
			if err := viper.MergeConfig(file); err != nil {
				return err
			}
		}
	}

	// Load user config.
	if userConfigFile := os.Getenv(envUserConfig); userConfigFile != "" {
		log.Printf("loading user configuration from %s", userConfigFile)
		var usercon userConfig
		data, err := ioutil.ReadFile(userConfigFile)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(data, &usercon); err != nil {
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

		data, err = yaml.Marshal(v)

		if err != nil {
			return err
		}
		r := bytes.NewReader(data)
		fmt.Printf("%s\n", data)

		if err := viper.MergeConfig(r); err != nil {
			return err
		}
	}

	return nil
}
