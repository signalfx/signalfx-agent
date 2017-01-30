package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// LoadYamlConfig parses a YAML file into an in-memory representation.
func LoadYamlConfig(configFile string) (*AppConfig, error) {
	config := NewCollectdConfig()
	data, err := ioutil.ReadFile(configFile)

	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %s", err)
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal main config: %s", err)
	}

	log.Infof("Loaded main config:\n%+v", config)

	var plugins []IPlugin

	for _, pluginConfig := range config.Plugins {
		yamlPluginType, ok := pluginConfig["plugin"]
		if !ok || yamlPluginType == nil {
			return nil, errors.New("plugin instance is missing the `plugin` key")
		}

		pluginType, ok := yamlPluginType.(string)
		if !ok {
			return nil, errors.New("plugin type not of type string")
		}

		// TODO: Check for name collissions.
		yamlPluginName, ok := pluginConfig["name"]
		if !ok || yamlPluginName == nil {
			return nil, errors.New("plugin instance is missing name `name` key")
		}
		pluginName, ok := yamlPluginName.(string)
		if !ok {
			return nil, errors.New("plugin name not of type string")
		}

		pluginStruct, err := NewPlugin(pluginType, pluginName)
		if err != nil {
			return nil, err
		}

		if err := loadPluginConfig(pluginConfig, pluginType, pluginStruct); err == nil {
			plugins = append(plugins, pluginStruct)
		} else {
			return nil, err
		}
	}

	return &AppConfig{AgentConfig: config, Plugins: plugins}, nil
}

// Load a plugin's configuration. If the configuration can't be loaded an error
// is returned.
func loadPluginConfig(config map[string]interface{}, plugin string, configStruct interface{}) error {
	log.Infof("Configuring %s plugin.", plugin)

	// We pluck the plugin configuration from the main config then marshal it
	// so that we can then unmarshal it into the plugin-specific struct.
	if out, err := yaml.Marshal(config); err == nil {
		// Entry could be "plugin:" with nothing else which would be null.
		if config != nil {
			if err := yaml.Unmarshal(out, configStruct); err != nil {
				return fmt.Errorf("failed to unmarshal %s configuration: %s", plugin, err)
			}
			return nil
		}
		log.Debugf("%s plugin had an empty configuration.", plugin)
		return nil
	}

	// Not including out in the log since it could contain sensitive data.
	return fmt.Errorf("failed to marshal user configuration for %s plugin", plugin)
}
