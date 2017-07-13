package plugins

import (
	"github.com/spf13/viper"
)

// PluginFactory creates an unconfigured instance of a plugin
type PluginFactory func() interface{}

// PluginFactories is a mapping of plugin names available to a creation function
var PluginFactories = map[string]PluginFactory{}

// Register registers a plugin
func Register(pluginType string, factory PluginFactory) {
	if _, ok := PluginFactories[pluginType]; ok {
		panic("plugin type '" + pluginType + "' already registered")
	}
	PluginFactories[pluginType] = factory
}

// MakePlugin creates a plugin from the registered factories.  Returns nil if
// plugin type was not registered.
func MakePlugin(pluginType string) interface{} {
	factory, ok := PluginFactories[pluginType]
	if !ok {
		return nil
	}
	return factory()
}

// Configurable can be implemented by plugin instances if they want to receive
// configuration
type Configurable interface {
	Configure(*viper.Viper) error
}

// Shutdownable can be implemented by plugin instances if they need to clean up
// resources on shutdown
type Shutdownable interface {
	Shutdown()
}

// Plugin is a wrapper around the actual plugin instances.  This is better than
// embedding the plugin type (as was previously done) because it keeps the
// plugin implementations as simple as possible.
type Plugin struct {
	name     string
	_type    string
	Instance interface{}
}

// Name of the plugin (based on key in config file)
func (p *Plugin) Name() string {
	return p.name
}

// Type of plugin, defined by the plugin itself
func (p *Plugin) Type() string {
	return p._type
}

// Configure the plugin if it is configurable
func (p *Plugin) Configure(config *viper.Viper) error {
	if c, ok := p.Instance.(Configurable); ok {
		return c.Configure(config)
	}
	return nil
}

// Shutdown the plugin if it supports that
func (p *Plugin) Shutdown() {
	if s, ok := p.Instance.(Shutdownable); ok {
		s.Shutdown()
	}
}
