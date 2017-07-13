package plugins

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/signalfx/neo-agent/config"
	"github.com/spf13/viper"
)

// Manager of plugins
type Manager struct {
	plugins    map[string]*Plugin
	configLock sync.Mutex
}

func addEnvVarOverrides(pluginName string, c *viper.Viper) {
	// This allows a configuration variable foo.bar to be overridable by
	// SFX_FOO_BAR=value.
	c.AutomaticEnv()
	c.SetEnvKeyReplacer(config.EnvReplacer)
	c.SetEnvPrefix(strings.ToUpper(
		fmt.Sprintf("%s_plugins_%s", config.EnvPrefix, pluginName)))
}

// Lock instance
func (m *Manager) Lock() {
	m.configLock.Lock()
}

// Unlock instance
func (m *Manager) Unlock() {
	m.configLock.Unlock()
}

// Load an agent using configuration file
func (m *Manager) Load() (map[string]*Plugin, error) {
	m.Lock()
	defer m.Unlock()

	if m.plugins == nil {
		m.plugins = make(map[string]*Plugin)
	}

	// Load plugins.
	pluginsConfig := viper.GetStringMap("plugins")

	m.shutdownUnconfiguredPlugins(pluginsConfig)

	for pluginName := range pluginsConfig {
		conf := viper.Sub("plugins." + pluginName)

		pluginType := conf.GetString("plugin")
		if len(pluginType) < 1 {
			log.Printf("Plugin %s missing plugin name value", pluginName)
			continue
		}

		pluginInst, err := m.ensurePluginExists(pluginName, pluginType)
		if err != nil {
			log.Printf("%s", err)
			continue
		}

		log.Printf("Configuring plugin %s (type %s)", pluginName, pluginType)

		addEnvVarOverrides(pluginName, conf)

		err = pluginInst.Configure(conf)
		if err != nil {
			log.Printf("Error configuring plugin %s: %s", pluginName, err)
			continue
		}
	}

	return m.plugins, nil
}

func (m *Manager) shutdownUnconfiguredPlugins(config map[string]interface{}) {
	for name, plugin := range m.plugins {
		if _, ok := config[name]; !ok {
			log.Printf("Plugin '%s' is no longer configured -- shutting down", name)
			plugin.Shutdown()
		}
	}
}

func (m *Manager) ensurePluginExists(name string, _type string) (*Plugin, error) {
	if inst, ok := m.plugins[name]; ok {
		return inst, nil
	}

	inst := MakePlugin(_type)
	if inst == nil {
		return nil, fmt.Errorf("Unknown plugin type: %s; did you register it "+
		"with RegisterFactory in the plugin module's init hook?", _type)
	}

	plugin := &Plugin{
		name: name,
		_type: _type,
		Instance: inst,
	}

	m.plugins[name] = plugin
	return plugin, nil
}

// Shutdown all plugins
func (m *Manager) Shutdown() {
	for name, plugin := range m.plugins {
		log.Printf("Shutting down plugin %s", name)
		plugin.Shutdown()
	}
}
