package config

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	"github.com/signalfx/neo-agent/core/config/stores"
)

// LoadConfigFromContent transforms yaml to a Config struct
func LoadConfigFromContent(fileContent []byte, metaStore *stores.MetaStore) (*Config, error) {
	config := &Config{}

	if err := defaults.Set(config); err != nil {
		panic(fmt.Sprintf("Config defaults are wrong types: %s", err))
	}

	err := yaml.Unmarshal(fileContent, config)
	if err != nil {
		return nil, err
	}

	return config.Initialize(metaStore)
}
