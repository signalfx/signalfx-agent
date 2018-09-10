// +build !windows

package rabbitmq

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/utils"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/rabbitmq"

// MONITOR(collectd/rabbitmq): Monitors an instance of RabbitMQ using the
// [collectd RabbitMQ Python
// Plugin](https://github.com/signalfx/collectd-rabbitmq).
//
// See the [integration
// doc](https://github.com/signalfx/integrations/tree/master/collectd-rabbitmq)
// for more information.
//
// **Note that you must individually enable each of the five `collect*` options
// to get metrics pertaining to those facets of a RabbitMQ instance.  If none
// of them are enabled, no metrics will be sent.**

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			python.Monitor{
				MonitorCore: pyrunner.New("sfxcollectd"),
			},
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	python.CoreConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host              string `yaml:"host" validate:"required"`
	Port              uint16 `yaml:"port" validate:"required"`
	// The name of the particular RabbitMQ instance.  Can be a Go template
	// using other config options. This will be used as the `plugin_instance`
	// dimension.
	BrokerName         string `yaml:"brokerName" default:"{{.host}}-{{.port}}"`
	CollectChannels    bool   `yaml:"collectChannels"`
	CollectConnections bool   `yaml:"collectConnections"`
	CollectExchanges   bool   `yaml:"collectExchanges"`
	CollectNodes       bool   `yaml:"collectNodes"`
	CollectQueues      bool   `yaml:"collectQueues"`
	HTTPTimeout        int    `yaml:"httpTimeout"`
	VerbosityLevel     string `yaml:"verbosityLevel"`
	Username           string `yaml:"username" validate:"required"`
	Password           string `yaml:"password" validate:"required" neverLog:"true"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in python
func (m *Monitor) Configure(conf *Config) error {
	conf.ModuleName = "rabbitmq"
	conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "rabbitmq")}
	conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	conf.PluginConfig = map[string]interface{}{
		"Host":               conf.Host,
		"Port":               conf.Port,
		"BrokerName":         conf.BrokerName,
		"Username":           conf.Username,
		"Password":           conf.Password,
		"CollectChannels":    conf.CollectChannels,
		"CollectConnections": conf.CollectConnections,
		"CollectExchanges":   conf.CollectExchanges,
		"CollectNodes":       conf.CollectNodes,
		"CollectQueues":      conf.CollectQueues,
	}

	// fill optional values into python plugin config map

	if conf.HTTPTimeout > 0 {
		conf.PluginConfig["HTTPTimeout"] = conf.HTTPTimeout
	}

	if conf.VerbosityLevel != "" {
		conf.PluginConfig["VerbosityLevel"] = conf.VerbosityLevel
	}

	// the python runner's templating system does not convert to map first
	// this requires TitleCase template values.  For BrokerName we accept
	// either upper or lower case values.  Converting the map to yaml
	// and explicitly rendering the BrokerName will allow for upper or lower
	// casing.
	mp, err := utils.ConvertToMapViaYAML(conf)
	if err != nil {
		return err
	}
	brokerName, err := collectd.RenderValue(conf.BrokerName, mp)
	if err != nil {
		return err
	}
	conf.PluginConfig["BrokerName"] = brokerName

	return m.Monitor.Configure(conf)
}
