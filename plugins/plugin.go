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

// NewPlugin constructor
func NewPlugin(name string, config *viper.Viper) (Plugin, error) {
	if config == nil {
		return Plugin{}, errors.New("config cannot be nil")
	}
	return Plugin{name, config}, nil
}

// String name of plugin
func (plugin *Plugin) String() string {
	return plugin.name
}
