package config

import (
	"context"
	"fmt"
	"os"
	"regexp"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/structtags"
	log "github.com/sirupsen/logrus"
)

type ConfigLoad struct {
	Config *Config
	Error  error
}

// LoadConfig handles loading the main config file and recursively rendering
// any dynamic values in the config.
func LoadConfig(ctx context.Context, configPath string) (<-chan ConfigLoad, error) {
	loads := make(chan ConfigLoad)

	configFileChanges, err := sources.StartReadingConfig(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %v", configPath, err)
	}

	go func() {
		dynamicProvider := sources.NewDynamicValueProvider()

		for {
			config, err := doSingleLoad(ctx, configFileChanges, dynamicProvider)
			loads <- ConfigLoad{config, err}
		}
	}()

	return loads, nil
}

func doSingleLoad(ctx context.Context, configFileChanges <-chan []byte, dynamicProvider *sources.DynamicValueProvider) (*Config, error) {
	// We can have changes either in the dynamic values or the
	// config file itself.  If the config file changes, we have to
	// recreate the dynamic value watcher since it is configured
	// from the config file.
	for {
		select {
		case configYAML := <-configFileChanges:
			err := dynamicProvider.ReadDynamicValues(ctx, configYAML)
			if err != nil {
				return nil, err
			}
		case finalYAML := <-dynamicProvider.Changes():
			config, err := loadYAML(finalYAML)
			if err != nil {
				return nil, err
			}
			return config, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func loadYAML(fileContent []byte) (*Config, error) {
	config := &Config{}

	preprocessedContent := preprocessConfig(fileContent)

	err := yaml.UnmarshalStrict(preprocessedContent, config)
	if err != nil {
		return nil, utils.YAMLErrorWithContext(preprocessedContent, err)
	}

	if err := defaults.Set(config); err != nil {
		panic(fmt.Sprintf("Config defaults are wrong types: %s", err))
	}

	if err := structtags.CopyTo(config); err != nil {
		panic(fmt.Sprintf("Error copying configs to fields: %v", err))
	}

	return config.initialize()
}

var envVarRE = regexp.MustCompile(`\${\s*([\w-]+?)\s*}`)

// Hold all of the envvars so that when they are sanitized from the proc we can
// still get to them when we need to rerender config
var envVarCache = make(map[string]string)

var envVarWhitelist = map[string]bool{
	"MY_NODE_NAME": true,
}

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

			if !envVarWhitelist[envvar] {
				os.Unsetenv(envvar)
			}
		}

		return []byte(val)
	})
}
