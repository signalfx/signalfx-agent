package config

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// LoadPluginConfig loads a plugin's configuration. If the configuration can't
// be loaded an error is returned.
func LoadPluginConfig(config map[string]interface{}, plugin string, configStruct interface{}) error {
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
