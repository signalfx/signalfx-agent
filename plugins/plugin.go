package plugins

import (
	"errors"

	"github.com/spf13/viper"
)

// Plugin type
type Plugin struct {
	name   string
	Config *viper.Viper
}

// IPlugin plugin interface
type IPlugin interface {
	Name() string
	Start() error
	Stop()
	String() string
}

// NewPlugin constructor
func NewPlugin(name string, config *viper.Viper) (Plugin, error) {
	if config == nil {
		return Plugin{}, errors.New("config cannot be nil")
	}
	return Plugin{name, config}, nil
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
