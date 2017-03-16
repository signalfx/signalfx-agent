package config

import (
	"log"
	"os"
	"strings"

	"github.com/signalfx/neo-agent/watchers"
	"github.com/spf13/viper"
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
)

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer = strings.NewReplacer(".", "_", "-", "_")
)

// WatchForChanges watches for changes to configuration files and reloads on change
func WatchForChanges(watcher *watchers.PollingWatcher, configfile string) {
	// Watch base config and merged config for changes. If either changes reload
	// viper config.
	configFiles := append(getMergeConfigs(), configfile)
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

	return nil
}
