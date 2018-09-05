// Package config contains configuration structures and related helper logic for all
// agent components.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	set "gopkg.in/fatih/set.v0"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"
	"github.com/signalfx/signalfx-agent/internal/core/common/constants"
	"github.com/signalfx/signalfx-agent/internal/core/config/sources"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

// Config is the top level config struct for configurations that are common to all platoforms
type Config struct {
	// The access token for the org that should receive the metrics emitted by
	// the agent.
	SignalFxAccessToken string `yaml:"signalFxAccessToken" neverLog:"true" validate:"required"`
	// The URL of SignalFx ingest server.  Can be overridden if using the Metric Proxy.
	IngestURL string `yaml:"ingestUrl" default:"https://ingest.signalfx.com"`
	// The SignalFx API base URL
	APIURL string `yaml:"apiUrl" default:"https://api.signalfx.com"`
	// The hostname that will be reported as the `host` dimension. If blank,
	// this will be auto-determined by the agent based on a reverse lookup of
	// the machine's IP address.
	Hostname string `yaml:"hostname"`
	// If true (the default), and the `hostname` option is not set, the
	// hostname will be determined by doing a reverse DNS query on the IP
	// address that is returned by querying for the bare hostname.  This is
	// useful in cases where the hostname reported by the kernel is a short
	// name.
	UseFullyQualifiedHost *bool `yaml:"useFullyQualifiedHost"`
	// Our standard agent model is to collect metrics for services running on
	// the same host as the agent.  Therefore, host-specific dimensions (e.g.
	// `host`, `AWSUniqueId`, etc) are automatically added to every datapoint
	// that is emitted from the agent by default.  Set this to true if you are
	// using the agent primarily to monitor things on other hosts.  You can set
	// this option at the monitor level as well.
	DisableHostDimensions bool `yaml:"disableHostDimensions" default:"false"`
	// How often to send metrics to SignalFx.  Monitors can override this
	// individually.
	IntervalSeconds int `yaml:"intervalSeconds" default:"10"`
    // Dimensions (key:value pairs) that will be added to every datapoint emitted by the agent.
    // To specify that all metrics should be high-resolution, add the dimension `sf_hires:1`
	GlobalDimensions map[string]string `yaml:"globalDimensions" default:"{}"`
	// Whether to send the machine-id dimension on all host-specific datapoints
	// generated by the agent.  This dimension is derived from the Linux
	// machine-id value.
	SendMachineID bool `yaml:"sendMachineID"`
	// A list of observers to use (see observer config)
	Observers []ObserverConfig `yaml:"observers" default:"[]" neverLog:"omit"`
	// A list of monitors to use (see monitor config)
	Monitors []MonitorConfig `yaml:"monitors" default:"[]" neverLog:"omit"`
	// Configuration of the datapoint/event writer
	Writer WriterConfig `yaml:"writer"`
	// Log configuration
	Logging LogConfig `yaml:"logging" default:"{}"`
	// Configuration of the managed collectd subprocess
	Collectd CollectdConfig `yaml:"collectd" default:"{}"`
	// A list of metric filters
	MetricsToExclude []MetricFilter `yaml:"metricsToExclude" default:"[]"`
	// (**NOT FUNCTIONAL**) Whether to enable the Python sub-agent ("neopy")
	// that can directly use DataDog and Collectd Python plugins.  This is not
	// the same as Collectd's Python plugin, which is always enabled.
	PythonEnabled bool `yaml:"pythonEnabled" default:"false"`

	DiagnosticsServerPath string `yaml:"-"`
	// The path where the agent will create a named pipe and serve diagnostic output (windows only)
	DiagnosticsServerNamedPipePath string `yaml:"diagnosticsNamedPipePath" default:"\\\\.\\pipe\\signalfx-agent-diagnostics" copyTo:"DiagnosticsServerPath,GOOS=windows"`
	// The path where the agent will create UNIX socket and serve diagnostic output (linux only)
	DiagnosticsSocketPath string `yaml:"diagnosticsSocketPath" default:"/var/run/signalfx-agent/diagnostics.sock" copyTo:"DiagnosticsServerPath,GOOS=!windows"`

	InternalMetricsServerPath string `yaml:"-"`
	// The path where the agent will create a named pipe that serves internal
	// metrics (used by the internal-metrics monitor) (windows only)
	InternalMetricsServerNamedPipePath string `yaml:"internalMetricsNamedPipePath" default:"\\\\.\\pipe\\signalfx-agent-internal-metrics" copyTo:"InternalMetricsServerPath,GOOS=windows"`
	// The path where the agent will create a socket that serves internal
	// metrics (used by the internal-metrics monitor) (linux only)
	InternalMetricsSocketPath string `yaml:"internalMetricsSocketPath" default:"/var/run/signalfx-agent/internal-metrics.sock" copyTo:"InternalMetricsServerPath,GOOS=!windows"`

	// Enables Go pprof endpoint on port 6060 that serves profiling data for
	// development
	EnableProfiling bool `yaml:"profiling" default:"false"`
	// Path to the directory holding the agent dependencies.  This will
	// normally be derived automatically. Overrides the envvar
	// SIGNALFX_BUNDLE_DIR if set.
	BundleDir string `yaml:"bundleDir"`
	// This exists purely to give the user a place to put common yaml values to
	// reference in other parts of the config file.
	Scratch interface{} `yaml:"scratch" neverLog:"omit"`
	// Configuration of remote config stores
	Sources sources.SourceConfig `yaml:"configSources"`
}

func (c *Config) initialize() (*Config, error) {
	c.setupEnvironment()

	if err := c.validate(); err != nil {
		return nil, errors.WithMessage(err, "configuration is invalid")
	}

	err := c.propagateValuesDown()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Setup envvars that will be used by collectd to use the bundled dependencies
// instead of looking to the normal system paths.
func (c *Config) setupEnvironment() {
	if c.BundleDir == "" {
		c.BundleDir = os.Getenv(constants.BundleDirEnvVar)
	}
	if c.BundleDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			panic("Cannot determine agent executable path, cannot continue")
		}
		c.BundleDir, err = filepath.Abs(filepath.Join(filepath.Dir(exePath), ".."))
		if err != nil {
			panic("Cannot determine absolute path of executable parent dir " + exePath)
		}
		os.Setenv(constants.BundleDirEnvVar, c.BundleDir)
	}

	os.Setenv("LD_LIBRARY_PATH", filepath.Join(c.BundleDir, "lib"))
	os.Setenv("JAVA_HOME", filepath.Join(c.BundleDir, "jvm/java-8-openjdk-amd64"))
	os.Setenv("PYTHONHOME", c.BundleDir)
}

// Validate everything that we can about the main config
func (c *Config) validate() error {
	if c.SignalFxAccessToken == "" {
		return fmt.Errorf("signalFxAccessToken must be set")
	}

	if _, err := url.Parse(c.IngestURL); err != nil {
		return errors.WithMessage(err, fmt.Sprintf("%s is not a valid ingest URL", c.IngestURL))
	}

	return c.Collectd.Validate()
}

// Send values from the top of the config down to nested configs that might
// need them
func (c *Config) propagateValuesDown() error {
	filterSet, err := makeFilterSet(c.MetricsToExclude)
	if err != nil {
		return err
	}

	ingestURL, err := url.Parse(c.IngestURL)
	if err != nil {
		panic("IngestURL was supposed to be validated already")
	}

	apiURL, err := url.Parse(c.APIURL)
	if err != nil {
		panic("apiUrl was supposed to be validated already")
	}

	for i := range c.Monitors {
		if err := c.Monitors[i].Init(); err != nil {
			return errors.WithMessage(err, fmt.Sprintf("Could not initialize monitor %s", c.Monitors[i].Type))
		}
	}

	c.Collectd.IntervalSeconds = utils.FirstNonZero(c.Collectd.IntervalSeconds, c.IntervalSeconds)
	c.Collectd.BundleDir = c.BundleDir

	c.Writer.IngestURL = ingestURL
	c.Writer.APIURL = apiURL
	c.Writer.Filter = filterSet
	c.Writer.SignalFxAccessToken = c.SignalFxAccessToken
	c.Writer.GlobalDimensions = c.GlobalDimensions

	return nil
}

// CustomConfigurable should be implemented by config structs that have the
// concept of generic other config that is initially deserialized into a
// map[string]interface{} to be later transformed to another form.
type CustomConfigurable interface {
	ExtraConfig() map[string]interface{}
}

// LogConfig contains configuration related to logging
type LogConfig struct {
	// Valid levels include `debug`, `info`, `warn`, `error`.  Note that
	// `debug` logging may leak sensitive configuration (e.g. passwords) to the
	// agent output.
	Level string `yaml:"level" default:"info"`
	// TODO: Support log file output and other log targets
}

// LogrusLevel returns a logrus log level based on the configured level in
// LogConfig.
func (lc *LogConfig) LogrusLevel() *log.Level {
	if lc.Level != "" {
		level, err := log.ParseLevel(lc.Level)
		if err != nil {
			log.WithFields(log.Fields{
				"level": lc.Level,
			}).Error("Invalid log level")
			return nil
		}
		return &level
	}
	return nil
}

var validCollectdLogLevels = set.NewNonTS("debug", "info", "notice", "warning", "err")

// CollectdConfig high-level configurations
type CollectdConfig struct {
	// If you won't be using any collectd monitors, this can be set to true to
	// prevent collectd from pre-initializing
	DisableCollectd bool `yaml:"disableCollectd" default:"false"`
	// How many read intervals before abandoning a metric. Doesn't affect much
	// in normal usage.
	// See [Timeout](https://collectd.org/documentation/manpages/collectd.conf.5.shtml#timeout_iterations).
	Timeout int `yaml:"timeout" default:"40"`
	// Number of threads dedicated to executing read callbacks. See
	// [ReadThreads](https://collectd.org/documentation/manpages/collectd.conf.5.shtml#readthreads_num)
	ReadThreads int `yaml:"readThreads" default:"5"`
	// Number of threads dedicated to writing value lists to write callbacks.
	// This should be much less than readThreads because writing is batched in
	// the write_http plugin that writes back to the agent.
	// See [WriteThreads](https://collectd.org/documentation/manpages/collectd.conf.5.shtml#writethreads_num).
	WriteThreads int `yaml:"writeThreads" default:"2"`
	// The maximum numbers of values in the queue to be written back to the
	// agent from collectd.  Since the values are written to a local socket
	// that the agent exposes, there should be almost no queuing and the
	// default should be more than sufficient. See
	// [WriteQueueLimitHigh](https://collectd.org/documentation/manpages/collectd.conf.5.shtml#writequeuelimithigh_highnum)
	WriteQueueLimitHigh int `yaml:"writeQueueLimitHigh" default:"500000"`
	// The lowest number of values in the collectd queue before which metrics
	// begin being randomly dropped.  See
	// [WriteQueueLimitLow](https://collectd.org/documentation/manpages/collectd.conf.5.shtml#writequeuelimitlow_lownum)
	WriteQueueLimitLow int `yaml:"writeQueueLimitLow" default:"400000"`
	// Collectd's log level -- info, notice, warning, or err
	LogLevel string `yaml:"logLevel" default:"notice"`
	// A default read interval for collectd plugins.  If zero or undefined,
	// will default to the global agent interval.  Some collectd python
	// monitors do not support overridding the interval at the monitor level,
	// but this setting will apply to them.
	IntervalSeconds int `yaml:"intervalSeconds" default:"0"`
	// The local IP address of the server that the agent exposes to which
	// collectd will send metrics.  This defaults to an arbitrary address in
	// the localhost subnet, but can be overridden if needed.
	WriteServerIPAddr string `yaml:"writeServerIPAddr" default:"127.9.8.7"`
	// The port of the agent's collectd metric sink server.  If set to zero
	// (the default) it will allow the OS to assign it a free port.
	WriteServerPort uint16 `yaml:"writeServerPort" default:"0"`
	// This is where the agent will write the collectd config files that it
	// manages.  If you have secrets in those files, consider setting this to a
	// path on a tmpfs mount.  The files in this directory should be considered
	// transient -- there is no value in editing them by hand.  If you want to
	// add your own collectd config, see the collectd/custom monitor.
	ConfigDir string `yaml:"configDir" default:"/var/run/signalfx-agent/collectd"`

	// The following are propagated from the top-level config
	BundleDir            string `yaml:"-"`
	HasGenericJMXMonitor bool   `yaml:"-"`
	// Assigned by manager, not by user
	InstanceName string `yaml:"-"`
	// A hack to allow custom collectd to easily specify a single monitorID via
	// query parameter
	WriteServerQuery string `yaml:"-"`
}

// Validate the collectd specific config
func (cc *CollectdConfig) Validate() error {
	if !validCollectdLogLevels.Has(cc.LogLevel) {
		return fmt.Errorf("Invalid collectd log level %s.  Valid choices are %v",
			cc.LogLevel, validCollectdLogLevels)
	}

	return nil
}

// Hash calculates a unique hash value for this config struct
func (cc *CollectdConfig) Hash() uint64 {
	hash, err := hashstructure.Hash(cc, nil)
	if err != nil {
		log.WithError(err).Error("Could not get hash of CollectdConfig struct")
		return 0
	}
	return hash
}

// WriteServerURL is the local address served by the agent where collect should
// write datapoints
func (cc *CollectdConfig) WriteServerURL() string {
	return fmt.Sprintf("http://%s:%d/", cc.WriteServerIPAddr, cc.WriteServerPort)
}

// InstanceConfigDir is the directory underneath the ConfigDir that is specific
// to this collectd instance.
func (cc *CollectdConfig) InstanceConfigDir() string {
	return filepath.Join(cc.ConfigDir, cc.InstanceName)
}

// ConfigFilePath returns the path where collectd should render its main config
// file.
func (cc *CollectdConfig) ConfigFilePath() string {
	return filepath.Join(cc.InstanceConfigDir(), "collectd.conf")
}

// ManagedConfigDir returns the dir path where all monitor config should go.
func (cc *CollectdConfig) ManagedConfigDir() string {
	return filepath.Join(cc.InstanceConfigDir(), "managed_config")
}

// StoreConfig holds configuration related to config stores (e.g. filesystem,
// zookeeper, etc)
type StoreConfig struct {
	OtherConfig map[string]interface{} `yaml:",inline,omitempty" default:"{}"`
}

// ExtraConfig returns generic config as a map
func (sc *StoreConfig) ExtraConfig() map[string]interface{} {
	return sc.OtherConfig
}

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer   = strings.NewReplacer(".", "_", "-", "_")
	configTimeout = 10 * time.Second
)
