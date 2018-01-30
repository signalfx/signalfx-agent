package config

import (
	"fmt"
	"os"
	"regexp"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	log "github.com/sirupsen/logrus"
)

// LoadConfigFromContent transforms yaml to a Config struct
func LoadConfigFromContent(fileContent []byte) (*Config, error) {
	config := &Config{}

	preprocessedContent := preprocessConfig(fileContent)

	err := yaml.UnmarshalStrict(preprocessedContent, config)
	if err != nil {
		log.WithError(err).Error("Could not unmarshal config file")
		return nil, err
	}

	if err := defaults.Set(config); err != nil {
		panic(fmt.Sprintf("Config defaults are wrong types: %s", err))
	}

	return config.initialize()
}

var envVarRE = regexp.MustCompile(`\${\s*([\w-]+?)\s*}`)

// Hold all of the envvars so that when they are sanitized from the proc we can
// still get to them when we need to rerender config
var envVarCache = make(map[string]string)

// Replaces envvar syntax with the actual envvars
func preprocessConfig(content []byte) []byte {
	return envVarRE.ReplaceAllFunc(content, func(bs []byte) []byte {
		parts := envVarRE.FindSubmatch(bs)
		envvar := string(parts[1])

		val, ok := envVarCache[envvar]

		if !ok {
			val = os.Getenv(envvar)
			envVarCache[envvar] = val

			log.WithFields(log.Fields{
				"envvar": envvar,
			}).Debug("Sanitizing envvar from agent")

			os.Unsetenv(envvar)
		}

		return []byte(val)
	})
}
