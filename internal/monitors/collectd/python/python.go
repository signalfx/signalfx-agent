// Package pyrunner contains a monitor that runs Collectd Python plugins
// directly in a subprocess.  It uses the logic in internal/monitors/pyrunner
// to do most of the work of managing a Python subprocess and doing the
// configuration/shutdown calls.

package python

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/mailru/easyjson"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	mpCollectd "github.com/signalfx/metricproxy/protocol/collectd"
	"github.com/signalfx/metricproxy/protocol/collectd/format"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/pyrunner"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils/collectdutil"
	log "github.com/sirupsen/logrus"
)

const messageTypeValueList pyrunner.MessageType = 100

const monitorType = "collectd/python"

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			MonitorCore: pyrunner.New("sfxcollectd"),
		}
	}, &Config{})
}

// PyConfig is an interface for passing in Config structs derrived from the Python Config struct
type PyConfig interface {
	config.MonitorCustomConfig
	PythonConfig() *Config
}

// MONITOR(collectd/python): This monitor runs arbitrary collectd Python
// plugins directly, apart from collectd.  It implements a mock collectd Python
// interface that supports most, but not all, of the real collectd.

// Config for the Collectd Python runner
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`

	// Host will be filled in by auto-discovery if this monitor has a discovery
	// rule.  It can then be used under pluginConfig by the template
	// `{{.Host}}`
	Host string `yaml:"host"`
	// Port will be filled in by auto-discovery if this monitor has a discovery
	// rule.  It can then be used under pluginConfig by the template
	// `{{.Port}}`
	Port uint16 `yaml:"port"`

	// Corresponds to the ModuleName option in collectd-python
	ModuleName string `yaml:"moduleName" json:"moduleName"`
	// Corresponds to a set of ModulePath options in collectd-python
	ModulePaths []string `yaml:"modulePaths" json:"modulePaths"`
	// This is a yaml form of the collectd config.
	PluginConfig map[string]interface{} `yaml:"pluginConfig" json:"pluginConfig"`
	// A set of paths to [types.db files](https://collectd.org/documentation/manpages/types.db.5.shtml)
	// that are needed by your plugin.  If not specified, the runner will use
	// the global collectd types.db file.
	TypesDBPaths []string `yaml:"typesDBPaths" json:"typesDBPaths"`
}

// PythonConfig returns the embedded python.Config struct from the interface
func (c *Config) PythonConfig() *Config {
	return c
}

// Monitor that runs collectd python plugins directly
type Monitor struct {
	*pyrunner.MonitorCore

	Output types.Output
}

// Configure starts the subprocess and configures the plugin
func (m *Monitor) Configure(conf PyConfig) error {
	// get the python config from the supplied config
	pyconf := conf.PythonConfig()
	if len(pyconf.TypesDBPaths) == 0 {
		pyconf.TypesDBPaths = append(pyconf.TypesDBPaths,
			filepath.Join(os.Getenv(constants.BundleDirEnvVar), "plugins/collectd/types.db"))
	}

	for k := range pyconf.PluginConfig {
		if v, ok := pyconf.PluginConfig[k].(string); ok {
			if v == "" {
				continue
			}

			template, err := template.New("nested").Parse(v)
			if err != nil {
				m.Logger().WithError(err).Error("Could not parse value '%s' as template", v)
				continue
			}

			out := bytes.Buffer{}
			// fill in any templates with the whole config struct passed into this method
			err = template.Option("missingkey=error").Execute(&out, conf)
			if err != nil {
				m.Logger().WithFields(log.Fields{
					"template": v,
					"error":    err,
					"context":  spew.Sdump(conf),
				}).Error("Could not render nested config template")
				continue
			}

			var result interface{} = out.String()
			if i, err := strconv.Atoi(result.(string)); err == nil {
				result = i
			}

			pyconf.PluginConfig[k] = result
		}
	}

	return m.MonitorCore.ConfigureInPython(pyconf, func(dataReader pyrunner.MessageReceiver) {
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

func (m *Monitor) handleMessage(msgType pyrunner.MessageType, payloadReader io.Reader) error {
	switch msgType {
	case messageTypeValueList:
		var valueList collectdformat.JSONWriteFormat
		if err := easyjson.UnmarshalFromReader(payloadReader, &valueList); err != nil {
			return err
		}

		dps := make([]*datapoint.Datapoint, 0)
		events := make([]*event.Event, 0)

		collectdutil.ConvertWriteFormat((*mpCollectd.JSONWriteFormat)(&valueList), &dps, &events)

		for i := range dps {
			m.Output.SendDatapoint(dps[i])
		}
		for i := range events {
			m.Output.SendEvent(events[i])
		}

	case pyrunner.MessageTypeLog:
		return m.HandleLogMessage(payloadReader)
	default:
		return fmt.Errorf("Unknown message type received %d", msgType)
	}

	return nil
}
