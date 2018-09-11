// +build !windows

package etcd

import (
	"os"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
)

const monitorType = "collectd/etcd"

// MONITOR(collectd/etcd): Monitors an etcd key/value store.
//
// See https://github.com/signalfx/integrations/tree/master/collectd-etcd and
// https://github.com/signalfx/collectd-etcd

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
	// An arbitrary name of the etcd cluster to make it easier to group
	// together and identify instances.
	ClusterName       string `yaml:"clusterName" validate:"required"`
	SSLKeyFile        string `yaml:"sslKeyFile"`
	SSLCertificate    string `yaml:"sslCertificate"`
	SSLCACerts        string `yaml:"sslCACerts"`
	SkipSSLValidation bool   `yaml:"skipSSLValidation"`
	EnhancedMetrics   bool   `yaml:"enhancedMetrics"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	python.Monitor
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	conf.PluginConfig = map[string]interface{}{
		"Host":                conf.Host,
		"Port":                conf.Port,
		"Interval":            conf.IntervalSeconds,
		"Cluster":             conf.ClusterName,
		"ssl_cert_validation": conf.SkipSSLValidation,
		"EnhancedMetrics":     conf.EnhancedMetrics,
	}
	if conf.ModuleName == "" {
		conf.ModuleName = "etcd_plugin"
	}
	if len(conf.ModulePaths) == 0 {
		conf.ModulePaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "etcd")}
	}
	if len(conf.TypesDBPaths) == 0 {
		conf.TypesDBPaths = []string{filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins", "collectd", "types.db")}
	}
	if conf.SSLKeyFile != "" {
		conf.PluginConfig["ssl_keyfile"] = conf.SSLKeyFile
	}
	if conf.SSLCertificate != "" {
		conf.PluginConfig["ssl_certificate"] = conf.SSLCertificate
	}
	if conf.SSLCACerts != "" {
		conf.PluginConfig["ssl_ca_certs"] = conf.SSLCACerts
	}
	return m.Monitor.Configure(conf)
}
