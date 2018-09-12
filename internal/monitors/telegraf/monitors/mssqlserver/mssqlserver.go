package mssqlserver

import (
	"context"
	"fmt"
	"time"

	telegrafInputs "github.com/influxdata/telegraf/plugins/inputs"
	telegrafPlugin "github.com/influxdata/telegraf/plugins/inputs/sqlserver"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/accumulator"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/emitter/baseemitter"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/sqlserver"

// MONITOR(telegraf/sqlserver): This monitor reports metrics about microsoft sql servers.
// This monitor is based on the telegraf sqlserver plugin.  More information about the telegraf plugin
// can be found [here](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/sqlserver).
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: telegraf/sqlserver
//    host: hostname
//    port: 1433
//    userID: sa
//    password: P@ssw0rd!
//    appName: signalfxagent
//    azureDB: true
//    excludeQuery:
//     - PerformanceCounters
//     # - WaitStatsCategorized
//     # - DatabaseIO
//     # - DatabaseProperties
//     # - CPUHistory
//     # - DatabaseSize
//     # - DatabaseStats
//     # - MemoryClerk
//     # - VolumeSpace
//     # - PerformanceMetrics
// ```
//

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"true"`
	Host                 string `yaml:"host" validate:"required" default:"."`
	Port                 uint16 `yaml:"port" validate:"required" default:"1433"`
	// UserID used to access the SQL Server instance.
	UserID string `yaml:"userID"`
	// Password used to access the SQL Server instance.
	Password string `yaml:"password" neverLog:"true"`
	// The app name used by the monitor when connecting to the SQLServer.
	AppName string `yaml:"appName" default:"signalfxagent"`
	// The version of queries to use when accessing the cluster
	// Please refer to the telegraf [documentation](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/sqlserver)
	// for more information.
	QueryVersion int `yaml:"queryVersion" default:"2"`
	// Whether the database is an azure database or not.
	AzureDB bool `yaml:"azureDB"`
	// Queries to exclude possible values are `PerformanceCounters`, `WaitStatsCategorized`,
	// `DatabaseIO`, `DatabaseProperties`, `CPUHistory`, `DatabaseSize`, `DatabaseStats`, `MemoryClerk`
	// `VolumeSpace`, `PerformanceMetrics`.
	ExcludeQuery []string `yaml:"excludedQueries"`
	// Log level to use when accessing the database
	Log uint `yaml:"log" default:"1"`
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

// fetch the factory used to generate the perf counter plugin
var factory = telegrafInputs.Inputs["sqlserver"]

// Configure the monitor and kick off metric syncing
func (m *Monitor) Configure(conf *Config) error {
	plugin := factory().(*telegrafPlugin.SQLServer)

	server := fmt.Sprintf("Server=%s;Port=%d;", conf.Host, conf.Port)

	if conf.UserID != "" {
		server = fmt.Sprintf("%sUser Id=%s;", server, conf.UserID)
	}
	if conf.Password != "" {
		server = fmt.Sprintf("%sPassword=%s;", server, conf.Password)
	}
	if conf.AppName != "" {
		server = fmt.Sprintf("%sapp name=%s;", server, conf.AppName)
	}
	server = fmt.Sprintf("%slog=%d;", server, conf.Log)

	plugin.Servers = []string{server}
	plugin.QueryVersion = conf.QueryVersion
	plugin.AzureDB = conf.AzureDB
	plugin.ExcludeQuery = conf.ExcludeQuery

	// create the accumulator
	ac := accumulator.NewAccumulator(baseemitter.NewEmitter(m.Output, logger))

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		if err := plugin.Gather(ac); err != nil {
			logger.Error(err)
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
