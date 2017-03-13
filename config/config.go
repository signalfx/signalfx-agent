package config

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	// DefaultInterval is used if not configured
	DefaultInterval = 10
	// DefaultPipeline is used if not configured
	DefaultPipeline = "docker"
	// EnvPrefix is the environment variable prefix
	EnvPrefix = "SFX"

	envMergeConfig = "SFX_MERGE_CONFIG"
)

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer = strings.NewReplacer(".", "_", "-", "_")
)

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

// WatchForChanges watches for changes to configuration files
func WatchForChanges(watcher *fsnotify.Watcher, configfile string, reload func() error) error {
	// Watch base config and merged config for changes. If either changes reload
	// viper config.
	configFileSet := map[string]bool{}
	configFiles := append(getMergeConfigs(), configfile)

	for _, configFile := range configFiles {
		configDir, configName := path.Split(configFile)
		// Remove trailing slash so ReadLink works.
		configDir = path.Clean(configDir)

		// Have to dereference symlink so that entry in configFileSet will match
		// the path from fsnotify.
		if linkDir, err := os.Readlink(configDir); err == nil {
			configDir = linkDir
		}

		configFileSet[path.Join(configDir, configName)] = true
		log.Printf("watching for changes to %s in %s", configFile, configDir)
		// Have to monitor directory in case file doesn't exist.
		if err := watcher.Add(configDir); err != nil {
			return err
		}
	}

	go func() {
		for {
			select {
			case err := <-watcher.Errors:
				if err == nil {
					return
				}
				log.Printf("file watch error: %s", err)
			case event := <-watcher.Events:
				if event.Op == 0 {
					return
				}
				// We're monitoring the whole directory so make sure the change
				// maps to a file we're interested in.
				if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Write) != 0 && configFileSet[event.Name] {
					log.Printf("config file %s changed", event.Name)
					if err := reload(); err != nil {
						log.Printf("error reloading configuration: %s", err)
					}
				}
			}
		}
	}()

	return nil
}

// Load loads the config from configfile as well as any merge files from
// environment variable
func Load(configfile string) error {
	viper.SetDefault("interval", DefaultInterval)
	viper.SetDefault("pipeline", DefaultPipeline)

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
