// +build !windows

package mysql

//go:generate collectd-template-to-go mysql.tmpl

import (
	"errors"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/mysql"

// MONITOR(collectd/mysql): Monitors a MySQL database server using collectd's
// [MySQL plugin](https://collectd.org/wiki/index.php/Plugin:MySQL).
//
// On Unix, MySQL programs treat the host name `localhost` specially, in a way
// that is likely different from what is expected compared to other
// network-based programs. For connections to `localhost`, MySQL programs
// attempt to connect to the local server by using a Unix socket file. To ensure
// that the client makes a TCP/IP connection to the local server specify a host
// name value of `127.0.0.1`, or the IP address or name of the local server.
//
// You have to specify each database you want to monitor individually under the
// `databases` key.  If you have a common authentication to all databases being
// monitored, you can specify that in the top-level `username`/`password`
// options, otherwise they can be specified at the database level.
//
// **Note:** The MySQL monitor supports MySQL versions 5.x or later.
//
// Sample YAML configuration:
//
// ```
// monitors:
//  - type: collectd/mysql
//    host: 127.0.0.1
//    port: 3306
//    databases:
//      - name: dbname
//      - name: securedb
//        username: admin
//        password: s3cr3t
//    username: dbuser
//    password: passwd
// ```

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			*collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Database configures a particular MySQL database
type Database struct {
	Name     string `yaml:"name" validate:"required"`
	Username string `yaml:"username"`
	Password string `yaml:"password" neverLog:"true"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	Host string `yaml:"host" validate:"required"`
	Port uint16 `yaml:"port" validate:"required"`
	Name string `yaml:"name"`
	// A list of databases along with optional authentication credentials.
	Databases []Database `yaml:"databases" validate:"required"`
	// These credentials serve as defaults for all databases if not overridden
	Username string `yaml:"username"`
	Password string `yaml:"password" neverLog:"true"`
	// A SignalFx extension to the plugin that allows us to disable the normal
	// behavior of the MySQL collectd plugin where the `host` dimension is set
	// to the hostname of the MySQL database server.  When `false` (the
	// recommended and default setting), the globally configured `hostname`
	// config is used instead.
	ReportHost bool `yaml:"reportHost"`
}

// Validate will check the config for correctness.
func (c *Config) Validate() error {
	if len(c.Databases) == 0 {
		return errors.New("You must specify at least one database for MySQL")
	}

	for _, db := range c.Databases {
		if db.Username == "" && c.Username == "" {
			return errors.New("Username is required for MySQL monitoring")
		}
	}
	return nil
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (am *Monitor) Configure(conf *Config) error {
	return am.SetConfigurationAndRun(conf)
}
