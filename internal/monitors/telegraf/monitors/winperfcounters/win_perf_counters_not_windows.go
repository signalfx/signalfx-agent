// +build !windows

package winperfcounters

import (
	"fmt"

	"github.com/influxdata/telegraf/plugins/inputs/win_perf_counters"
	"github.com/ulule/deepcopier"
)

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	telegraf_plugin := &win_perf_counters.Win_PerfCounters{}
	deepcopier.Copy(conf).To(telegraf_plugin)
	fmt.Printf("telegraf_plugin: %v", telegraf_plugin)
	return nil
}
