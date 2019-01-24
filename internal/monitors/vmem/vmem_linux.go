// +build linux

package vmem

import (
	"bytes"
	"context"
	"io/ioutil"
	"path"
	"strconv"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/hostfs"
)

var cumulativeCounters = map[string]string{
	"pgpgin":     "vmpage_io.memory.in",
	"pgpgout":    "vmpage_io.memory.out",
	"pswpin":     "vmpage_io.swap.in",
	"pswpout":    "vmpage_io.swap.out",
	"pgmajfault": "vmpage_faults.majflt",
	"pgfault":    "vmpage_faults.minflt",
}

var gauges = map[string]string{
	"nr_free_pages": "vmpage_number.free_pages",
	"nr_mapped":     "vmpage_number.mapped",
	"nr_shmem":      "vmpage_number.shmem_pmdmapped",
}

func (m *Monitor) parseFile(contents []byte) {
	data := bytes.Fields(contents)
	max := len(data)
	for i, key := range data {
		// vmstat file structure is (key, value)
		// so every even index is a key and every odd index is the value
		if i%2 == 0 && i+1 < max {
			keyStr := string(key)

			metricType := datapoint.Gauge
			metricName, ok := gauges[keyStr]
			if !ok {
				metricName, ok = cumulativeCounters[keyStr]
				metricType = datapoint.Counter
			}

			// build and emit the metric if there's a metric name
			if ok {
				val, err := strconv.ParseInt(string(data[i+1]), 10, 64)
				if err != nil {
					logger.Errorf("failed to parse value for metric %s", metricName)
					continue
				}
				m.Output.SendDatapoint(datapoint.New(metricName, map[string]string{"plugin": monitorType}, datapoint.NewIntValue(val), metricType, time.Time{}))
			}
		}
	}
}

// Configure and run the monitor on linux
func (m *Monitor) Configure(conf *Config) (err error) {
	logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	vmstatPath := path.Join(hostfs.HostProc(), "vmstat")

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		contents, err := ioutil.ReadFile(vmstatPath)
		if err != nil {
			logger.WithError(err).Errorf("unable to load vmstat file from path '%s'", vmstatPath)
			return
		}
		m.parseFile(contents)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}
