package hana

import (
	"context"
	dbsql "database/sql"
	"fmt"
	"strconv"
	"sync"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/sql"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for the postgresql monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host                 string            `yaml:"host"`
	Port                 uint16            `yaml:"port"`
	ConnectionString     string            `yaml:"connectionString"`
	Params               map[string]string `yaml:"params"`
	LogQueries           bool              `yaml:"logQueries"`
}

func (c *Config) connStr() (template string, port string, err error) {
	var host = "localhost"
	if c.Host != "" {
		host = c.Host
	}
	port = "443"
	if c.Port != 0 {
		port = strconv.Itoa(int(c.Port))
	}
	connStr := "hdb://{{.username}}:{{.password}}@" + host + ":" + port + "?TLSInsecureSkipVerify=false&TLSServerName=" + host
	if c.ConnectionString != "" {
		connStr = c.ConnectionString
	}
	template, err = utils.RenderSimpleTemplate(connStr, c.Params)
	fmt.Println(template)
	return
}

// Monitor that collects SAP Hana stats
type Monitor struct {
	sync.Mutex

	Output types.FilteringOutput
	ctx    context.Context
	cancel context.CancelFunc
	conf   *Config

	database *dbsql.DB

	serverMonitor *sql.Monitor
}

// Configure the monitor and kick off metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.conf = conf
	m.ctx, m.cancel = context.WithCancel(context.Background())

	connStr, _, err := conf.connStr()
	if err != nil {
		return fmt.Errorf("could not render connectionString template: %v", err)
	}
	m.database, err = dbsql.Open("hdb", connStr)
	if err != nil {
		return fmt.Errorf("Failed to open database: %v", err)
	}

	m.serverMonitor, err = m.monitorServer()
	if err != nil {
		m.database.Close()
		return fmt.Errorf("could not monitor Hana server: %v", err)
	}

	return nil
}

func (m *Monitor) monitorServer() (*sql.Monitor, error) {
	sqlMon := &sql.Monitor{Output: m.Output.Copy()}

	connStr, _, err := m.conf.connStr()
	if err != nil {
		return nil, err
	}

	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    m.conf.MonitorConfig,
		ConnectionString: connStr,
		DBDriver:         "hdb",
		Queries:          defaultServerQueries,
		LogQueries:       m.conf.LogQueries,
	})
}

// Shutdown this monitor and the nested sql ones
func (m *Monitor) Shutdown() {
	m.Lock()
	defer m.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	if m.database != nil {
		_ = m.database.Close()
	}

	if m.serverMonitor != nil {
		m.serverMonitor.Shutdown()
	}

}
