package config

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources/file"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

// LoadConfig handles loading the main config file and recursively rendering
// any dynamic values in the config.  If watchInterval is 0, the config will be
// loaded once and sent to the returned channel, after which the channel will
// be closed.  Otherwise, the returned channel will remain open and will be
// sent any config updates.
func LoadConfig(ctx context.Context, configPath string, watchInterval time.Duration) (<-chan *Config, error) {
	fileSource := file.New(watchInterval)
	shouldWatch := uint64(watchInterval) != 0

	configYAML, configFileChanges, err := sources.ReadConfig(configPath, fileSource, ctx.Done(), shouldWatch)
	if err != nil {
		return nil, errors.WithMessage(err, "Could not read config file "+configPath)
	}

	dynamicValueCtx, cancelDynamic := context.WithCancel(ctx)
	finalYAML, dynamicChanges, err := sources.ReadDynamicValues(configYAML, fileSource, dynamicValueCtx.Done(), shouldWatch)
	if err != nil {
		cancelDynamic()
		return nil, err
	}

	config, err := loadYAML(finalYAML)
	if err != nil {
		cancelDynamic()
		return nil, err
	}

	// Give it enough room to hold the initial config load.
	loads := make(chan *Config, 1)

	loads <- config

	if shouldWatch {
		go func() {
			for {
				// We can have changes either in the dynamic values or the
				// config file itself.  If the config file changes, we have to
				// recreate the dynamic value watcher since it is configured
				// from the config file.
				select {
				case configYAML = <-configFileChanges:
					cancelDynamic()

					dynamicValueCtx, cancelDynamic = context.WithCancel(ctx)

					finalYAML, dynamicChanges, err = sources.ReadDynamicValues(configYAML, fileSource, dynamicValueCtx.Done(), true)
					if err != nil {
						log.WithError(err).Error("Could not read dynamic values in config after change")
						time.Sleep(5 * time.Second)
						continue
					}

					config, err := loadYAML(finalYAML)
					if err != nil {
						log.WithError(err).Error("Could not parse config after change")
						continue
					}

					loads <- config
				case finalYAML = <-dynamicChanges:
					config, err := loadYAML(finalYAML)
					if err != nil {
						log.WithError(err).Error("Could not parse config after change")
						continue
					}
					loads <- config
				case <-ctx.Done():
					cancelDynamic()
					return
				}
			}
		}()
	} else {
		cancelDynamic()
		close(loads)
	}
	return loads, nil
}

func loadYAML(fileContent []byte) (*Config, error) {
	config := &Config{}

	preprocessedContent := preprocessConfig(fileContent)

	err := yaml.UnmarshalStrict(preprocessedContent, config)
	if err != nil {
		// Provide some context about where the parse error occurred
		for _, e := range err.(*yaml.TypeError).Errors {
			line := utils.ParseLineNumberFromYAMLError(e)
			context := string(preprocessedContent)
			if line != 0 {
				lines := strings.Split(context, "\n")
				context = strings.Join(lines[int(math.Max(float64(line-5), 0)):line], "\n")
				context += "\n^^^^^^^\n"
				context += strings.Join(lines[line:int(math.Min(float64(line+5), float64(len(lines))))], "\n")
			}
			log.Errorf("Could not unmarshal config file:\n\n%s\n\n%s\n", context, err.Error())
		}
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
