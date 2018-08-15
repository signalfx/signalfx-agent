package processlist

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const (
	monitorType = "processlist"
	version     = "0.0.30"
)

// MONITOR(processlist): This monitor reports processlist information for Windows
// Hosts.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
//  - type: processlist
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false"`
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

func compressBytes(in []byte) (buf bytes.Buffer, err error) {
	compressor := zlib.NewWriter(&buf)
	_, err = compressor.Write(in)
	_ = compressor.Close()
	return
}

func toTime(secs float64) (response string) {
	minutes := int(secs / 60)
	seconds := int(math.Mod(secs, 60.0))
	sec := seconds
	dec := (seconds - sec) * 100
	response = fmt.Sprintf("%02d:%02d.%02d", minutes, sec, dec)
	return
}

// Monitor for Utilization
type Monitor struct {
	Output types.Output
	cancel func()
}

// Configure configures the monitor and starts collecting on the configured interval
func (m *Monitor) Configure(conf *Config) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("currently this monitor only supports windows")
	}

	// create contexts for managing the the plugin loop
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	utils.RunOnInterval(
		ctx,
		func() {
			// get the process list
			processList, err := ProcessList()
			if err != nil {
				logger.Error(err)
				return
			}

			// escape and compress the process list
			escapedBytes := bytes.Replace(processList.Bytes(), []byte{byte('\\')}, []byte{byte('\\'), byte('\\')}, -1)
			cbytes, err := compressBytes(escapedBytes)
			if err != nil {
				logger.Error(err)
				return
			}

			// format and emit the top-info event
			message := fmt.Sprintf("{\"t\":\"%s\",\"v\":\"%s\"}", base64.StdEncoding.EncodeToString(cbytes.Bytes()), version)
			m.Output.SendEvent(
				&event.Event{
					EventType:  "objects.top-info",
					Category:   event.AGENT,
					Dimensions: map[string]string{},
					Properties: map[string]interface{}{
						"message": message,
					},
				},
			)
		},
		time.Duration(conf.IntervalSeconds)*time.Second,
	)
	return nil
}
