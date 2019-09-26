package jmx

import (
	"fmt"
	"path/filepath"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/subproc/signalfx/java"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
		return &Monitor{
			Monitor: java.NewMonitorCore(),
		}
	}, &Config{})
}

// Config for the JMX Monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Host will be filled in by auto-discovery if this monitor has a discovery
	// rule.
	Host string `yaml:"host" json:"host,omitempty"`
	// Port will be filled in by auto-discovery if this monitor has a discovery
	// rule.
	Port uint16 `yaml:"port" json:"port,omitempty"`
	// The service URL for the JMX RMI endpoint.  If empty it will be filled in
	// with values from `host` and `port` using a standard template:
	// service:jmx:rmi:///jndi/rmi://<host>:<port>/jmxrmi.  If overridden,
	// `host` and `port` will have no effect.
	ServiceURL string `yaml:"serviceURL" json:"serviceURL"`
	// A literal Groovy script that generates datapoints from JMX MBeans.  See
	// the top-level `jmx` monitor doc for more information on how to write
	// this script. You can put the Groovy script in a separate file and refer
	// to it here with the [remote config
	// reference](https://docs.signalfx.com/en/latest/integrations/agent/remote-config.html)
	// `{"#from": "/path/to/file.groovy", raw: true}`, or you can put it
	// straight in YAML by using the `|` heredoc syntax.
	GroovyScript string `yaml:"groovyScript" json:"groovyScript"`
	// Username for JMX authentication, if applicable.
	Username string `yaml:"username" json:"username"`
	// Password for JMX autentication, if applicable.
	Password string `yaml:"password" json:"password" neverLog:"true"`
}

type Monitor struct {
	*java.Monitor
}

func (m *Monitor) Configure(conf *Config) error {
	serviceURL := conf.ServiceURL
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("service:jmx:rmi:///jndi/rmi://%s:%d/jmxrmi", conf.Host, conf.Port)
	}
	return m.Monitor.Configure(&java.Config{
		MonitorConfig: conf.MonitorConfig,
		Host:          conf.Host,
		Port:          conf.Port,
		JarFilePath:   filepath.Join(conf.BundleDir(), "lib/jmx-monitor.jar"),
		CustomConfig: map[string]interface{}{
			"serviceURL":   serviceURL,
			"groovyScript": conf.GroovyScript,
			"username":     conf.Username,
			"password":     conf.Password,
		},
	})
}
