package postgresql

import (
	"context"
	dbsql "database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/core/dpfilters"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/sql"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/signalfx/signalfx-agent/pkg/utils/filter"
	"github.com/sirupsen/logrus"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for the postgresql monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
	// The "master" database to which the agent first connects to query the
	// list of databases available in the server.  This database should be
	// accessible to the user specified with `connectionString` and `params`
	// below, and that user should have permission to query `pg_database`.  If
	// you want to filter which databases are monitored, use the `databases`
	// option below.
	MasterDBName string `yaml:"masterDBName" default:"postgres"`

	// See
	// https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters.
	ConnectionString string `yaml:"connectionString"`
	// Parameters to the connection string that can be templated into the
	// connection string with the syntax `{{.key}}`.
	Params map[string]string `yaml:"params"`

	// List of databases to send database-specific metrics about.  If omitted, metrics about all databases will be sent.  This is an [overridable set](https://docs.signalfx.com/en/latest/integrations/agent/filtering.html#overridable-filters).
	Databases []string `yaml:"databases" default:"[\"*\"]"`

	// How frequently to poll for new/deleted databases in the DB server.
	// Defaults to the same as `intervalSeconds` if not set.
	DatabasePollIntervalSeconds int `yaml:"databasePollIntervalSeconds"`

	// If true, queries will be logged at the info level.
	LogQueries bool `yaml:"logQueries"`

	// The number of top queries to consider when publishing query-related metrics
	TopQueryLimit int `default:"10" yaml:"topQueryLimit"`
}

func (c *Config) connStr() (template string, port string, err error) {
	connStr := c.ConnectionString
	port = "5432"
	if c.Host != "" {
		connStr += " host=" + c.Host
	}
	if c.Port != 0 {
		connStr += fmt.Sprintf(" port=%d", c.Port)
		port = strconv.Itoa(int(c.Port))
	}
	template, err = utils.RenderSimpleTemplate(connStr, c.Params)
	return
}

// Monitor that collects postgresql stats
type Monitor struct {
	sync.Mutex

	Output types.FilteringOutput
	ctx    context.Context
	cancel context.CancelFunc
	conf   *Config

	database *dbsql.DB

	monitoredDBs       map[string]*sql.Monitor
	serverMonitor      *sql.Monitor
	statementsMonitor  *sql.Monitor
	replicationMonitor *sql.Monitor

	// name for execution time column determined by information schema for pg_stat_statement
	totalTimeColumn string
}

// Configure the monitor and kick off metric collection
func (m *Monitor) Configure(conf *Config) error {
	m.conf = conf
	m.ctx, m.cancel = context.WithCancel(context.Background())

	logger := logrus.WithFields(logrus.Fields{
		"monitorType": monitorMetadata.MonitorType,
		"host":        conf.Host,
		"port":        conf.Port,
	})

	queriesGroupEnabled := m.Output.HasEnabledMetricInGroup(groupQueries)
	replicationGroupEnabled := m.Output.HasEnabledMetricInGroup(groupReplication)

	connStr, port, err := conf.connStr()
	if err != nil {
		return fmt.Errorf("could not render connectionString template: %v", err)
	}
	m.Output.AddExtraDimension("postgres_port", port)

	var dbFilter filter.StringFilter
	if len(conf.Databases) > 0 {
		dbFilter, err = filter.NewOverridableStringFilter(conf.Databases)
		if err != nil {
			return fmt.Errorf("problem with databases filter: %v", err)
		}
	}

	databaseDatapointFilter, err := dpfilters.NewOverridable(nil, map[string][]string{
		"database?": conf.Databases,
	})
	if err != nil {
		return err
	}
	m.Output.AddDatapointExclusionFilter(dpfilters.Negate(databaseDatapointFilter))

	dbPollInterval := time.Duration(conf.IntervalSeconds) * time.Second
	if conf.DatabasePollIntervalSeconds != 0 {
		dbPollInterval = time.Duration(conf.DatabasePollIntervalSeconds) * time.Second
	}

	m.monitoredDBs = map[string]*sql.Monitor{}

	m.serverMonitor, err = m.monitorServer()
	if err != nil {
		m.database.Close()
		return fmt.Errorf("could not monitor postgresql server: %v", err)
	}

	if queriesGroupEnabled {
		m.statementsMonitor, err = m.monitorStatements()
		if err != nil {
			logger.WithError(err).Errorf("Could not monitor queries: %v", err)
		}
	}

	replicationStarted := false

	utils.RunOnInterval(m.ctx, func() {
		m.Lock()
		defer m.Unlock()

		// This means the monitor is shutdown
		if m.ctx.Err() != nil {
			return
		}

		if m.database == nil {
			m.database, err = dbsql.Open("postgres", connStr+" dbname="+m.conf.MasterDBName)
			if err != nil {
				logger.WithError(err).WithField("connStr", connStr).Error("Failed to open database")
				return
			}
		}

		if replicationGroupEnabled && !replicationStarted {
			rows, err := m.database.QueryContext(m.ctx, `select AURORA_VERSION();`)
			if err == nil {
				defer rows.Close()
				logger.Info("Aurora server detected, disabling replication monitor")
			} else {
				logger.Debug("Replication metrics enabled")
				m.replicationMonitor, err = m.monitorReplication()
				if err != nil {
					logger.WithError(err).Errorf("Could not monitor replication: %v", err)
				}
			}
			replicationStarted = true
		}

		databases, err := determineDatabases(m.ctx, m.database)
		if err != nil {
			logger.WithError(err).Error("Could not determine list of PostgreSQL databases")
		}

		dbSet := map[string]bool{}

		// Start monitoring any new databases
		for _, db := range databases {
			if dbFilter != nil && !dbFilter.Matches(db) {
				continue
			}

			dbSet[db] = true
			if _, ok := m.monitoredDBs[db]; !ok {
				mon, err := m.startMonitoringDatabase(db)
				if err != nil {
					logger.WithError(err).Errorf("Could not monitor database '%s'", db)
					continue
				}
				m.monitoredDBs[db] = mon
				logger.Infof("Now monitoring PostgreSQL database '%s'", db)
			}
		}

		// Stop monitoring any dbs that disappear.
		for name := range m.monitoredDBs {
			if !dbSet[name] {
				logger.Infof("No longer monitoring PostgreSQL database '%s'", name)
				m.monitoredDBs[name].Shutdown()
				delete(m.monitoredDBs, name)
			}
		}
	}, dbPollInterval)

	return nil
}

func (m *Monitor) startMonitoringDatabase(name string) (*sql.Monitor, error) {
	connStr, _, err := m.conf.connStr()
	if err != nil {
		return nil, err
	}

	connStr += " dbname=" + name

	sqlMon := &sql.Monitor{Output: m.Output.Copy()}
	sqlMon.Output.AddExtraDimension("database", name)

	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    m.conf.MonitorConfig,
		ConnectionString: connStr,
		DBDriver:         "postgres",
		Queries:          makeDefaultDBQueries(name),
		LogQueries:       m.conf.LogQueries,
	})
}

func determineDatabases(ctx context.Context, database *dbsql.DB) ([]string, error) {
	rows, err := database.QueryContext(ctx, `SELECT datname FROM pg_database WHERE datistemplate = false;`)
	if err != nil {
		return nil, err
	}
	if rows != nil {
		defer func() {
			_ = rows.Close()
		}()
	}

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (m *Monitor) determineTotalTimeColumn(connStr string) (string, error) {
	database, err := dbsql.Open("postgres", connStr)
	if err != nil {
		return "", fmt.Errorf("could not handle postgres database config: %w", err)
	}
	defer database.Close()

	rows, err := database.QueryContext(m.ctx, `SELECT column_name FROM information_schema.columns WHERE table_name='pg_stat_statements' and column_name SIMILAR TO 'total_(exec_|)time';`)
	if err != nil {
		return "", err
	}
	if rows != nil {
		defer func() {
			_ = rows.Close()
		}()
	}

	var totalTimeColumn string
	for rows.Next() { // there is only one resulting row
		if err := rows.Scan(&totalTimeColumn); err != nil {
			return "", err
		}
	}
	return totalTimeColumn, nil
}

func (m *Monitor) monitorServer() (*sql.Monitor, error) {
	sqlMon := &sql.Monitor{Output: m.Output.Copy()}

	connStr, _, err := m.conf.connStr()
	if err != nil {
		return nil, err
	}

	connStr += " dbname=" + m.conf.MasterDBName

	m.totalTimeColumn, err = m.determineTotalTimeColumn(connStr)
	if err != nil {
		return nil, err
	}

	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    m.conf.MonitorConfig,
		ConnectionString: connStr,
		DBDriver:         "postgres",
		Queries:          defaultServerQueries(m.totalTimeColumn),
		LogQueries:       m.conf.LogQueries,
	})
}

func (m *Monitor) monitorStatements() (*sql.Monitor, error) {
	sqlMon := &sql.Monitor{Output: m.Output.Copy()}

	connStr, _, err := m.conf.connStr()
	if err != nil {
		return nil, err
	}

	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    m.conf.MonitorConfig,
		ConnectionString: connStr + " dbname=" + m.conf.MasterDBName,
		DBDriver:         "postgres",
		Queries:          makeDefaultStatementsQueries(m.conf.TopQueryLimit, m.totalTimeColumn),
		LogQueries:       m.conf.LogQueries,
	})
}

func (m *Monitor) monitorReplication() (*sql.Monitor, error) {
	sqlMon := &sql.Monitor{Output: m.Output.Copy()}

	connStr, _, err := m.conf.connStr()
	if err != nil {
		return nil, err
	}

	return sqlMon, sqlMon.Configure(&sql.Config{
		MonitorConfig:    m.conf.MonitorConfig,
		ConnectionString: connStr + " dbname=" + m.conf.MasterDBName,
		DBDriver:         "postgres",
		Queries:          defaultReplicationQueries,
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

	for i := range m.monitoredDBs {
		m.monitoredDBs[i].Shutdown()
	}

	if m.serverMonitor != nil {
		m.serverMonitor.Shutdown()
	}

	if m.statementsMonitor != nil {
		m.statementsMonitor.Shutdown()
	}

	if m.replicationMonitor != nil {
		m.replicationMonitor.Shutdown()
	}
}
