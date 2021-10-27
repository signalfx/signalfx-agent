package hana

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/sql"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for the SAP Hana monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	ConnectionString     string
	Host                 string
	Username             string
	Password             string
	Port                 int
	LogQueries           bool
	MaxExpensiveQueries  int
}

// Monitor that collects SAP Hana stats
type Monitor struct {
	Output     types.FilteringOutput
	sqlMonitor *sql.Monitor
}

// Configure the monitor and kick off metric collection
func (m *Monitor) Configure(conf *Config) error {
	var err error
	m.sqlMonitor, err = configureSQLMonitor(
		m.Output.Copy(),
		conf.MonitorConfig,
		cfgToConnString(conf),
		conf.LogQueries,
		conf.MaxExpensiveQueries,
	)
	if err != nil {
		return fmt.Errorf("could not configure Hana SQL monitor: %v", err)
	}
	return nil
}

func cfgToConnString(c *Config) string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}
	return connString(c.Host, c.Port, c.Username, c.Password)
}

func connString(host string, port int, username string, password string) string {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 443
	}
	const format = "hdb://%s:%s@%s:%d?TLSInsecureSkipVerify=false&TLSServerName=%s"
	return fmt.Sprintf(format, username, password, host, port, host)
}

func configureSQLMonitor(output types.Output, monCfg config.MonitorConfig, connStr string, logQueries bool, maxExpensiveQueries int) (*sql.Monitor, error) {
	sqlMon := &sql.Monitor{Output: output}
	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    monCfg,
		ConnectionString: connStr,
		DBDriver:         "hdb",
		Queries:          queries(maxExpensiveQueries),
		LogQueries:       logQueries,
	})
}

func (m *Monitor) Shutdown() {
	m.sqlMonitor.Shutdown()
}
