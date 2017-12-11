package config

import (
	"fmt"
	"os"
	"regexp"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	"github.com/signalfx/neo-agent/core/config/stores"
	log "github.com/sirupsen/logrus"
)

// LoadConfigFromContent transforms yaml to a Config struct
func LoadConfigFromContent(fileContent []byte, metaStore *stores.MetaStore) (*Config, error) {
	config := &Config{}

	preprocessedContent := preprocessConfig(fileContent)
	log.Debugf("Pre: %s", preprocessedContent)

	err := yaml.UnmarshalStrict(preprocessedContent, config)
	if err != nil {
		log.WithError(err).Error("Could not unmarshal config file")
		return nil, err
	}

	if err := defaults.Set(config); err != nil {
		panic(fmt.Sprintf("Config defaults are wrong types: %s", err))
	}

	return config.initialize(metaStore)
}

var envVarRE = regexp.MustCompile(`\${\s*([\w-]+?)\s*}`)

// Replaces envvar syntax with the actual envvars
func preprocessConfig(content []byte) []byte {
	return envVarRE.ReplaceAllFunc(content, func(bs []byte) []byte {
		parts := envVarRE.FindSubmatch(bs)
		value := os.Getenv(string(parts[1]))
		log.WithFields(log.Fields{
			"envVarName": string(parts[1]),
			"value":      value,
		}).Info("Substituting envvar into config")

		return []byte(value)
	})
}
