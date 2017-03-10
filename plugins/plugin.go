package plugins

import (
	"errors"

	"github.com/spf13/viper"
)

// CreatePlugin is the function definition for creating a plugin
type CreatePlugin func(string, *viper.Viper) (IPlugin, error)

// Plugins is a mapping of plugin names available to a creation function
var Plugins = map[string]CreatePlugin{}

// Plugin type
type Plugin struct {
	name       string
	pluginType string
	Config     *viper.Viper
}

// IPlugin plugin interface
type IPlugin interface {
	Name() string
	Start() error
	Stop()
	Reload(config *viper.Viper) error
	String() string
	Type() string
}

// NewPlugin constructor
func NewPlugin(name, pluginType string, config *viper.Viper) (Plugin, error) {
	if config == nil {
		return Plugin{}, errors.New("config cannot be nil")
	}
	return Plugin{name, pluginType, config}, nil
}

// Name is name of plugin
func (plugin *Plugin) Name() string {
	return plugin.name
}

// String name of plugin
func (plugin *Plugin) String() string {
	return plugin.name
}

// Start default start (no-op)
func (plugin *Plugin) Start() error {
	return nil
}

// Stop default stop (no-op)
func (plugin *Plugin) Stop() {
}

// Type returns the plugin type
func (plugin *Plugin) Type() string {
	return plugin.pluginType
}

// Reload replaces the config with the newly loaded conifg. Plugins that need to
// do anything special should implement their own reload.
func (plugin *Plugin) Reload(config *viper.Viper) error {
	// Need to be careful about when this Reload function gets called so as to
	// avoid any data race issues with plugins. Right now reload is called
	// before an execution is run so most plugins won't have any code running in
	// other goroutines. If they do they should take care to synchronize the
	// config replacement.
	plugin.Config = config
	return nil
}

// Register registers a plugin
func Register(name string, create CreatePlugin) {
	if _, ok := Plugins[name]; ok {
		panic("plugin " + name + " already registered")
	}
	Plugins[name] = create
}
