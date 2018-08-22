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
var zlibCompressor = zlib.NewWriter(&bytes.Buffer{})

// Config for this monitor
type Config struct {
	config.MonitorConfig `singleInstance:"true" acceptsEndpoints:"false"`
}

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// compresses the given byte array
func compressBytes(in []byte) (buf bytes.Buffer, err error) {
	zlibCompressor.Reset(&buf)
	_, err = zlibCompressor.Write(in)
	_ = zlibCompressor.Close()
	return
}

// toTime returns the given seconds as a formatted string "min:sec.dec"
func toTime(secs float64) string {
	minutes := int(secs) / 60
	seconds := math.Mod(secs, 60.0)
	dec := math.Mod(seconds, 1.0) * 100
	return fmt.Sprintf("%02d:%02.f.%02.f", minutes, seconds, dec)
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
				logger.WithError(err).Error("Couldn't get process list")
				return
			}

			// escape and compress the process list
			escapedBytes := bytes.Replace(processList.Bytes(), []byte{byte('\\')}, []byte{byte('\\'), byte('\\')}, -1)
			compressedBytes, err := compressBytes(escapedBytes)
			if err != nil {
				logger.WithError(err).Error("Couldn't compress process list")
				return
			}

			// format and emit the top-info event
			message := fmt.Sprintf("{\"t\":\"%s\",\"v\":\"%s\"}", base64.StdEncoding.EncodeToString(compressedBytes.Bytes()), version)
			m.Output.SendEvent(
				&event.Event{
					EventType:  "objects.top-info",
					Category:   event.AGENT,
					Dimensions: map[string]string{},
					Properties: map[string]interface{}{
						"message": message,
					},
					Timestamp: time.Now(),
				},
			)
		},
		time.Duration(conf.IntervalSeconds)*time.Second,
	)
	return nil
}
