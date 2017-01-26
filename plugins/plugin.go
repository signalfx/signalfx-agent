package plugins

// Plugin type
type Plugin struct {
	name          string
	configuration map[string]string
}

// NewPlugin constructor
func NewPlugin(name string, configuration map[string]string) Plugin {
	return Plugin{name, configuration}
}

// String name of plugin
func (plugin *Plugin) String() string {
	return plugin.name
}

// GetConfig value from plugin configuration
func (plugin *Plugin) GetConfig(key string) (string, bool) {
	val, ok := plugin.configuration[key]
	return val, ok
}
