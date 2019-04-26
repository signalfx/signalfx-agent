// +build linux

package df

//go:generate collectd-template-to-go df.tmpl

import (
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd"
)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} {
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
	IgnoreSelected *bool `yaml:"ignoreSelected" default:"true"`

	// The filesystem types to include/exclude.
	FSTypes []string `yaml:"fsTypes" default:"[\"aufs\", \"overlay\", \"tmpfs\", \"proc\", \"sysfs\", \"nsfs\", \"cgroup\", \"devpts\", \"selinuxfs\", \"devtmpfs\", \"debugfs\", \"mqueue\", \"hugetlbfs\", \"securityfs\", \"pstore\", \"binfmt_misc\", \"autofs\"]"`

	// The mount paths to include/exclude, is interpreted as a regex if
	// surrounded by `/`.  Note that you need to include the full path as the
	// agent will see it, irrespective of the hostFSPath option.
	MountPoints    []string `yaml:"mountPoints" default:"[\"/^/var/lib/docker/containers/\", \"/^/var/lib/rkt/pods/\", \"/^/net//\", \"/^/smb//\"]"`
	ReportByDevice bool     `yaml:"reportByDevice" default:"false"`
	ReportInodes   bool     `yaml:"reportInodes" default:"false"`

	// If true percent based metrics will be reported.
	ValuesPercentage bool `yaml:"valuesPercentage" default:"false"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	collectd.MonitorCore
}

// GetExtraMetrics returns additional metrics to allow through.
func (c *Config) GetExtraMetrics() []string {
	var extraMetrics []string
	if c.ReportInodes {
		extraMetrics = append(extraMetrics, groupMetricsMap[groupReportInodes]...)
	}
	if c.ValuesPercentage {
		extraMetrics = append(extraMetrics, groupMetricsMap[groupValuesPercentage]...)
	}
	if c.ReportInodes && c.ValuesPercentage {
		extraMetrics = append(extraMetrics, []string{percentInodesFree, percentInodesReserved, percentInodesUsed}...)
	}
	return extraMetrics
}

// Configure configures and runs the plugin in collectd
func (m *Monitor) Configure(conf *Config) error {
	return m.SetConfigurationAndRun(conf)
}
