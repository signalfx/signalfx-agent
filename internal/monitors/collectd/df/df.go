package df

//go:generate collectd-template-to-go df.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

const monitorType = "collectd/df"

// MONITOR(collectd/df): Tracks free disk space on the host.

// GAUGE(gauge.df_complex.free): Free disk space in bytes
// GAUGE(gauge.df_complex.reserved): Reserved disk space in bytes
// GAUGE(gauge.df_complex.used): Used disk space in bytes
// GAUGE(gauge.df_inodes.free): Number of inodes that are free.
// GAUGE(gauge.df_inodes.reserved): Number of inodes reserved for the super user.
// GAUGE(gauge.df_inodes.used): Number of inodes that are used.

// GAUGE(gauge.percent_bytes.free): Free disk space on the file system,
// expressed as a percentage.

// GAUGE(gauge.percent_bytes.reserved): Reserved disk space on the filesystem,
// expressed as a percentage.

// GAUGE(gauge.percent_bytes.used): Used disk space on the file system,
// expressed as a percentage.

// GAUGE(gauge.percent_inodes.free): Free inodes on the file system, expressed
// as a percentage.

// GAUGE(gauge.percent_inodes.reserved): Reserved inodes on the file system,
// expressed as a percentage.

// GAUGE(gauge.percent_inodes.used): Used inodes on the file system, expressed
// as a percentage.

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: *collectd.NewMonitorCore(CollectdTemplate),
		}
	}, &Config{})
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `singleInstance:"true"`
	// Path to the root of the host filesystem.  Useful when running in a
	// container and the host filesystem is mounted in some subdirectory under
	// /.
	HostFSPath string `yaml:"hostFSPath"`

	// If true, the filesystems selected by `fsTypes` and `mountPoints` will be
	// excluded and all others included.
	IgnoreSelected *bool `yaml:"ignoreSelected"`

	// The filesystem types to include/exclude.
	FSTypes []string `yaml:"fsTypes" default:"[\"aufs\", \"overlay\", \"tmpfs\", \"proc\", \"sysfs\", \"nsfs\", \"cgroup\", \"devpts\", \"selinuxfs\", \"devtmpfs\", \"debugfs\", \"mqueue\", \"hugetlbfs\", \"securityfs\", \"pstore\", \"binfmt_misc\", \"autofs\"]"`

	// The mount paths to include/exclude, is interpreted as a regex if
	// surrounded by `/`.  Note that you need to include the full path as the
	// agent will see it, irrespective of the hostFSPath option.
	MountPoints    []string `yaml:"mountPoints" default:"[\"/^/var/lib/rkt/pods/\", \"/^/net//\", \"/^/smb//\"]"`
	ReportByDevice bool     `yaml:"reportByDevice" default:"false"`
	ReportInodes   bool     `yaml:"reportInodes" default:"false"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	t := true
	// Default to true.  Ideally don't have bools that default to true but this
	// one is pretty essential.
	if conf.IgnoreSelected == nil {
		conf.IgnoreSelected = &t
	}
	return m.SetConfigurationAndRun(conf)
}
