// Package signalfx contains a pyrunner implementation that works better with
// SignalFx datapoint than the collectd/python monitor, which must implement
// the collectd python interface.
package signalfx

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mailru/easyjson"
	"github.com/signalfx/com_signalfx_metrics_protobuf"
	"github.com/signalfx/gateway/protocol/signalfx"
	signalfxformat "github.com/signalfx/gateway/protocol/signalfx/format"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/sirupsen/logrus"
)

const messageTypeDatapointList pyrunner.MessageType = 200

const monitorType = "python-monitor"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &PyMonitor{
			MonitorCore: pyrunner.New("sfxmonitor"),
		}
	}, &Config{})
}

// PyConfig is an interface for passing in Config structs derrived from the Python Config struct
type PyConfig interface {
	config.MonitorCustomConfig
	PythonConfig() *Config
}

// CustomConfig is embedded in Config struct to catch all extra config to pass to Python
type CustomConfig map[string]interface{}

// Config specifies configurations that are specific to the individual python based monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	// Host will be filled in by auto-discovery if this monitor has a discovery
	// rule.
	Host string `yaml:"host" json:"host,omitempty"`
	// Port will be filled in by auto-discovery if this monitor has a discovery
	// rule.
	Port uint16 `yaml:"port" json:"port,omitempty"`
	// Path to the Python script that implements the monitoring logic.
	ScriptFilePath string `yaml:"scriptFilePath" json:"scriptFilePath"`
	// By default, the agent will use its bundled Python runtime (version 2.7).
	// If you wish to use a Python runtime that already exists on the system,
	// specify the full path to the `python` binary here, e.g.
	// `/usr/bin/python3`.
	PythonBinary string `yaml:"pythonBinary" json:"pythonBinary"`
	// The PYTHONPATH that will be used when importing the script specified at
	// `scriptFilePath`.  The directory of `scriptFilePath` will always be
	// included in the path.
	PythonPath   []string `yaml:"pythonPath" json:"pythonPath"`
	CustomConfig `yaml:",inline" json:"-" neverLog:"true"`
}

// MarshalJSON flattens out the CustomConfig provided by the user into a single
// map so that it is simpler to access config in Python.
func (c Config) MarshalJSON() ([]byte, error) {
	type ConfigX Config // prevent recursion
	b, err := json.Marshal(ConfigX(c))
	if err != nil {
		return nil, err
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	// Don't need this.
	delete(m, "OtherConfig")

	for k, v := range c.CustomConfig {
		m[k], err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(m)
}

// PyMonitor that runs collectd python plugins directly
type PyMonitor struct {
	*pyrunner.MonitorCore

	Output types.Output
}

// Configure starts the subprocess and configures the plugin
func (m *PyMonitor) Configure(conf *Config) error {
	runtimeConf := m.DefaultRuntimeConfig()
	if conf.PythonBinary != "" {
		runtimeConf.PythonBinary = conf.PythonBinary
		runtimeConf.PythonEnv = os.Environ()
	} else {
		// Pass down the default runtime binary to the Python script if it
		// needs it
		conf.PythonBinary = runtimeConf.PythonBinary
	}
	if len(conf.PythonPath) > 0 {
		runtimeConf.PythonEnv = append(runtimeConf.PythonEnv, "PYTHONPATH="+strings.Join(conf.PythonPath, ":"))
	}

	return m.MonitorCore.ConfigureInPython(conf, runtimeConf, func(dataReader pyrunner.MessageReceiver) {
		for {
			m.Logger().Debug("Waiting for messages")
			msgType, payloadReader, err := dataReader.RecvMessage()

			m.Logger().Debugf("Got message of type %d", msgType)

			// This is usually due to the pipe being closed
			if err != nil {
				m.Logger().WithError(err).Error("Could not receive messages")
				return
			}

			if m.ShutdownCalled() {
				return
			}

			if err := m.handleMessage(msgType, payloadReader); err != nil {
				m.Logger().WithError(err).Error("Could not handle message from Python")
				continue
			}
		}
	})
}

func (m *PyMonitor) handleMessage(msgType pyrunner.MessageType, payloadReader io.Reader) error {
	switch msgType {
	case messageTypeDatapointList:
		// The following is copied from github.com/signalfx/gateway
		var d signalfxformat.JSONDatapointV2
		if err := easyjson.UnmarshalFromReader(payloadReader, &d); err != nil {
			return err
		}
		dps := make([]*datapoint.Datapoint, 0, len(d))
		for metricType, datapoints := range d {
			if len(datapoints) > 0 {
				mt, ok := com_signalfx_metrics_protobuf.MetricType_value[strings.ToUpper(metricType)]
				if !ok {
					logrus.Error("Unknown metric type")
					continue
				}
				for _, jsonDatapoint := range datapoints {
					v, err := signalfx.ValueToValue(jsonDatapoint.Value)
					if err != nil {
						logrus.WithError(err).Error("Unable to get value for datapoint")
						continue
					}
					dp := datapoint.New(jsonDatapoint.Metric, jsonDatapoint.Dimensions, v, fromMT(com_signalfx_metrics_protobuf.MetricType(mt)), fromTs(jsonDatapoint.Timestamp))
					dps = append(dps, dp)
				}
			}
		}
		for i := range dps {
			m.Output.SendDatapoint(dps[i])
		}

	case pyrunner.MessageTypeLog:
		return m.HandleLogMessage(payloadReader)
	default:
		return fmt.Errorf("unknown message type received %d", msgType)
	}

	return nil
}

// Copied from github.com/signalfx/gateway
var fromMTMap = map[com_signalfx_metrics_protobuf.MetricType]datapoint.MetricType{
	com_signalfx_metrics_protobuf.MetricType_CUMULATIVE_COUNTER: datapoint.Counter,
	com_signalfx_metrics_protobuf.MetricType_GAUGE:              datapoint.Gauge,
	com_signalfx_metrics_protobuf.MetricType_COUNTER:            datapoint.Count,
}

func fromMT(mt com_signalfx_metrics_protobuf.MetricType) datapoint.MetricType {
	ret, exists := fromMTMap[mt]
	if exists {
		return ret
	}
	panic(fmt.Sprintf("Unknown metric type: %v\n", mt))
}

func fromTs(ts int64) time.Time {
	if ts > 0 {
		return time.Unix(0, ts*time.Millisecond.Nanoseconds())
	}
	return time.Now().Add(-time.Duration(time.Millisecond.Nanoseconds() * ts))
}
