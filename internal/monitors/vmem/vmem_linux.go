// +build linux

package vmem

import (
	"bytes"
	"context"
	"io/ioutil"
	"path"
	"strconv"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
	"github.com/signalfx/signalfx-agent/internal/utils/hostfs"
	"github.com/sirupsen/logrus"
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

func (m *Monitor) parseFileForDatapoints(contents []byte) []*datapoint.Datapoint {
	data := bytes.Fields(contents)
	max := len(data)
	dps := make([]*datapoint.Datapoint, 0, max)

	for i, key := range data {
		// vmstat file structure is (key, value)
		// so every even index is a key and every odd index is the value
		if i%2 == 0 && i+1 < max {
			metricType := datapoint.Gauge
			metricName, ok := gauges[string(key)]
			if !ok {
				metricName, ok = cumulativeCounters[string(key)]
				metricType = datapoint.Counter
			}

			// build and emit the metric if there's a metric name
			if ok {
				val, err := strconv.ParseInt(string(data[i+1]), 10, 64)
				if err != nil {
					m.logger.Errorf("failed to parse value for metric %s", metricName)
					continue
				}
				dps = append(dps, datapoint.New(metricName, map[string]string{"plugin": monitorType}, datapoint.NewIntValue(val), metricType, time.Time{}))
			}
		}
	}

	return dps
}

// Configure and run the monitor on linux
func (m *Monitor) Configure(conf *Config) (err error) {
	m.logger = logrus.WithField("monitorType", monitorType)
	m.logger.Warningf("'%s' monitor is in beta on this platform.  For production environments please use 'collectd/%s'.", monitorType, monitorType)

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	vmstatPath := path.Join(hostfs.HostProc(), "vmstat")

	// gather metrics on the specified interval
	utils.RunOnInterval(ctx, func() {
		contents, err := ioutil.ReadFile(vmstatPath)
		if err != nil {
			m.logger.WithError(err).Errorf("unable to load vmstat file from path '%s'", vmstatPath)
			return
		}
		m.Output.SendDatapoints(m.parseFileForDatapoints(contents)...)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}
