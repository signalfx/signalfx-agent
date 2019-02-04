package diskio

import (
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	log "github.com/sirupsen/logrus"
)

const monitorType = "disk-io"

// MONITOR(disk-io):
// This monitor reports I/O metrics about disks.
//
// On Linux hosts, this monitor relies on the `/proc` filesystem.
// If the underlying host's `/proc` file system is mounted somewhere other than
// /proc please specify the path using the top level configuration `procPath`.
//
// ```yaml
// procPath: /proc
// monitors:
//  - type: disk-io
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"false" acceptsEndpoints:"false"`
	// The devices to include/exclude. This is a
	// [filter set](https://github.com/signalfx/signalfx-agent/blob/master/docs/filtering.md#generic-filters).
	Disks []string `yaml:"disks" default:"[\"*\", \"!/^loop[0-9]+$/\", \"!/^dm-[0-9]+$/\"]"`
	// (Windows Only) The frequency that wildcards in counter paths should
	// be expanded and how often to refresh counters from configuration.
	// This is expressed as a duration.
	CountersRefreshInterval time.Duration `yaml:"counterRefreshInterval" default:"60s"`
	// (Windows Only) Print out the configurations that match available
	// performance counters.  This used for debugging.
	PrintValid bool `yaml:"printValid"`
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
