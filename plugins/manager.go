package plugins

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"time"

	"github.com/signalfx/neo-agent/config"
	"github.com/signalfx/neo-agent/watchers"
	"github.com/spf13/viper"
)

func configurePlugin(pluginName string, c *viper.Viper) {
	// This allows a configuration variable foo.bar to be overridable by
	// SFX_FOO_BAR=value.
	c.AutomaticEnv()
	c.SetEnvKeyReplacer(config.EnvReplacer)
	c.SetEnvPrefix(strings.ToUpper(
		fmt.Sprintf("%s_plugins_%s", config.EnvPrefix, pluginName)))
}

// configureWatching creates and/or updates the file watch list for a plugin
func configureWatching(plugin IPlugin, pluginConfig *viper.Viper, configLock *sync.Mutex) {
	watcher := plugin.Watcher()

	if !viper.GetBool("filewatching") {
		// Stop watcher if file watching was turned on before.
		if watcher != nil {
			watcher.Close()
			plugin.SetWatcher(nil)
		}
		return
	}

	pollingInterval := viper.GetFloat64("pollinginterval")
	if pollingInterval <= 0 {
		log.Printf("pollingInterval must greater than zero")
		return
	}

	duration := time.Duration(pollingInterval * float64(time.Second))

	watchFiles := plugin.GetWatchFiles(pluginConfig)
	watchDirs := plugin.GetWatchDirs(pluginConfig)

	if watcher == nil && (len(watchFiles) > 0 || len(watchDirs) > 0) {
		log.Printf("creating watcher for plugin %s", plugin.Name())
		watcher = watchers.NewPollingWatcher(func(changed []string) error {
			configLock.Lock()
			defer configLock.Unlock()

			log.Printf("%v changed for plugin %s", changed, plugin.Name())
			if err := plugin.Reload(pluginConfig); err != nil {
				log.Printf("error reloading plugin %s: %s", plugin.Name(), err)
			}
			return nil
		}, duration)
		plugin.SetWatcher(watcher)
		watcher.Start()

		// Need to reload plugin after starting watchers in case file changed
		// between plugin creation and starting watcher. XXX: This might be
		// better by having plugin constructors not initialize state but require
		// that be done in Start().
		if err := plugin.Reload(pluginConfig); err != nil {
			log.Printf("failed to reload plugin %s post-watch: %s", plugin.Name(), err)
		}
	}
	if watcher != nil {
		watcher.Watch(watchDirs, watchFiles)
	}
}

// Load an agent using configuration file
func Load(currentPlugins []IPlugin, configLock *sync.Mutex) ([]IPlugin, error) {
	// Load plugins.
	pluginsConfig := viper.GetStringMap("plugins")

	currentPluginSet := map[string]IPlugin{}
	for _, plugin := range currentPlugins {
		currentPluginSet[plugin.Name()] = plugin
	}

	var newPlugins []IPlugin
	var removedPlugins []IPlugin
	var reloadPlugins []IPlugin

	for pluginName := range pluginsConfig {
		pluginType := viper.GetString(fmt.Sprintf("plugins.%s.plugin", pluginName))

		if len(pluginType) < 1 {
			return nil, fmt.Errorf("plugin %s missing plugin key", pluginName)
		}

		if plugin := currentPluginSet[pluginName]; plugin != nil {
			// Already exists, just reload.
			reloadPlugins = append(reloadPlugins, plugin)
		} else {
			// New plugin
			if create, ok := Plugins[pluginType]; ok {
				log.Printf("configuring plugin %s (%s)", pluginType, pluginName)

				pluginConfig := viper.Sub("plugins." + pluginName)
				configurePlugin(pluginName, pluginConfig)
				pluginInst, err := create(pluginName, pluginConfig)
				if err != nil {
					return nil, err
				}
				configureWatching(pluginInst, pluginConfig, configLock)

				newPlugins = append(newPlugins, pluginInst)
			} else {
				return nil, fmt.Errorf("unknown plugin %s", pluginType)
			}
		}
	}

	// NOTE: By this point we can't return errors with an unmodified plugin list
	// as some new plugins may have been started. If there's an old plugin 'foo'
	// and we start a new 'foo' but return an old plugin set there might be two
	// foos running.

	// Find removed plugins (in loaded plugins but not the new config).
	for _, plugin := range currentPlugins {
		if pluginsConfig[plugin.Name()] == nil {
			removedPlugins = append(removedPlugins, plugin)
		}
	}

	// Stop removed plugins.
	for _, plugin := range removedPlugins {
		log.Printf("stopping plugin %s", plugin.Name())
		w := plugin.Watcher()
		if w != nil {
			log.Printf("stopping watcher for %s", plugin.Name())
			// Close is synchronous so the stopped plugin shouldn't get any more
			// file change notifications.
			w.Close()
		}
		plugin.Stop()
	}

	// Reload existing plugins.
	for _, plugin := range reloadPlugins {
		log.Printf("reloading plugin %s", plugin.Name())
		pluginConfig := viper.Sub("plugins." + plugin.Name())

		if pluginConfig == nil {
			log.Printf("%s plugin unexpectedly missing config", plugin.Name())
			continue
		}

		pluginType := pluginConfig.GetString("plugin")

		if pluginType != plugin.Type() {
			log.Printf("%s plugin is currently type %s but changed to %s, skipping",
				plugin.Name(), plugin.Type(), pluginType)
			continue
		}

		configurePlugin(plugin.Name(), pluginConfig)
		configureWatching(plugin, pluginConfig, configLock)

		if err := plugin.Reload(pluginConfig); err != nil {
			log.Printf("reloading %s plugin failed: %s", plugin.Name(), err)
		}
	}

	// Start new plugins.
	for _, plugin := range newPlugins {
		log.Printf("starting plugin %s", plugin.String())
		if err := plugin.Start(); err != nil {
			log.Printf("failed to start plugin %s: %s", plugin.String(), err)
		}
	}

	return append(reloadPlugins, newPlugins...), nil
}
