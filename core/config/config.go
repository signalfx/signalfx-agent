// Package config contains configuration structures and related helper logic for all
// agent components.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	set "gopkg.in/fatih/set.v0"

	fqdn "github.com/ShowMax/go-fqdn"
	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/core/config/stores"
	"github.com/signalfx/neo-agent/core/filters"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

// Config is the top level config struct that everything goes under
type Config struct {
	SignalFxAccessToken string `yaml:"signalFxAccessToken" neverLog:"true"`
	// The ingest URL for SignalFx, without the path
	IngestURL string `yaml:"ingestUrl" default:"https://ingest.signalfx.com"`
	// The SignalFx API base URL
	APIURL string `yaml:"apiUrl" default:"https://api.signalfx.com"`
	// The hostname that will be reported as the "host" dimension on metrics
	// for which host applies
	Hostname string `yaml:"hostname"`
	// How often to send metrics to SignalFx.  Monitors can override this
	// individually.
	IntervalSeconds int `yaml:"intervalSeconds" default:"10"`
	// Dimensions that will be automatically added to all metrics reported
	GlobalDimensions map[string]string `yaml:"globalDimensions" default:"{}"`
	Observers        []ObserverConfig  `yaml:"observers" default:"[]" neverLog:"omit"`
	Monitors         []MonitorConfig   `yaml:"monitors" default:"[]" neverLog:"omit"`
	Writer           WriterConfig      `yaml:"writer" default:"{}"`
	Logging          LogConfig         `yaml:"logging" default:"{}"`
	// Configure the underlying collectd daemon
	Collectd                  CollectdConfig `yaml:"collectd" default:"{}"`
	MetricsToExclude          []MetricFilter `yaml:"metricsToExclude" default:"[]"`
	ProcFSPath                string         `yaml:"procFSPath" default:"/proc"`
	PythonEnabled             bool           `yaml:"pythonEnabled" default:"false"`
	DiagnosticsSocketPath     string         `yaml:"diagnosticsSocketPath" default:"./run/signalfx.sock"`
	InternalMetricsSocketPath string         `yaml:"internalMetricsSocketPath" default:"./run/signalfx-agent-metrics.sock"`
	EnableProfiling           bool           `yaml:"profiling" default:"false"`
	// This exists purely to give the user a place to put common yaml values to
	// reference in other parts of the config file.
	Scratch interface{} `yaml:"scratch" neverLog:"omit"`
}

func (c *Config) setDefaultHostname() {
	host := fqdn.Get()
	if host == "unknown" || host == "localhost" {
		log.Info("Error getting fully qualified hostname, using plain hostname")

		var err error
		host, err = os.Hostname()
		if err != nil {
			log.Error("Error getting system simple hostname, cannot set hostname")
			return
		}
	}

	log.Infof("Using hostname %s", host)
	c.Hostname = host
}

func (c *Config) initialize(metaStore *stores.MetaStore) (*Config, error) {
	c.overrideFromEnv()

	c.setDefaultHostname()

	if err := c.validate(); err != nil {
		return nil, errors.Wrap(err, "configuration is invalid")
	}

	c.propagateValuesDown(metaStore)

	return c, nil
}

// Support overridding a few config options with envvars.  No need to allow
// everything to be overridden.
func (c *Config) overrideFromEnv() {
	c.SignalFxAccessToken = utils.FirstNonEmpty(c.SignalFxAccessToken, os.Getenv("SFX_ACCESS_TOKEN"))
	c.Hostname = utils.FirstNonEmpty(c.Hostname, os.Getenv("SFX_HOSTNAME"))
	c.IngestURL = utils.FirstNonEmpty(c.IngestURL, os.Getenv("SFX_INGEST_URL"))

	intervalSeconds, err := strconv.ParseInt(os.Getenv("SFX_INTERVAL_SECONDS"), 10, 32)
	if err != nil {
		c.IntervalSeconds = utils.FirstNonZero(c.IntervalSeconds, int(intervalSeconds))
	}
}

// Validate everything that we can about the main config
func (c *Config) validate() error {
	if c.SignalFxAccessToken == "" {
		return errors.New("signalFxAccessToken must be set")
	}

	if _, err := url.Parse(c.IngestURL); err != nil {
		return errors.Wrapf(err, "%s is not a valid ingest URL", c.IngestURL)
	}

	return c.Collectd.Validate()
}

func (c *Config) makeFilterSet() *filters.FilterSet {
	fs := make([]filters.Filter, 0)
	for _, mte := range c.MetricsToExclude {
		fs = append(fs, mte.MakeFilter())
	}

	return &filters.FilterSet{
		Filters: fs,
	}
}

// Send values from the top of the config down to nested configs that might
// need them
func (c *Config) propagateValuesDown(metaStore *stores.MetaStore) {
	filterSet := c.makeFilterSet()

	ingestURL, err := url.Parse(c.IngestURL)
	if err != nil {
		panic("IngestURL was supposed to be validated already")
	}

	apiURL, err := url.Parse(c.APIURL)
	if err != nil {
		panic("apiUrl was supposed to be validated already")
	}

	c.Collectd.Hostname = c.Hostname
	c.Collectd.Filter = filterSet
	c.Collectd.IngestURL = c.IngestURL
	c.Collectd.SignalFxAccessToken = c.SignalFxAccessToken
	c.Collectd.GlobalDimensions = c.GlobalDimensions
	c.Collectd.IntervalSeconds = utils.FirstNonZero(c.Collectd.IntervalSeconds, c.IntervalSeconds)

	// If the root mount namespace is mounted at ./hostfs we need to tell
	// collectd about it so that disk utilization metrics can be properly
	// stripped of this prefix in the df plugin in collectd.
	if hostFSPath, err := filepath.Abs("./hostfs"); err == nil {
		if _, err := os.Stat(hostFSPath); err == nil {
			c.Collectd.HostFSPath = hostFSPath
		}
	} else {
		log.Info("Could not find ./hostfs, assuming running in host's root mount namespace")
	}

	for i := range c.Monitors {
		c.Monitors[i].CollectdConf = &c.Collectd
		c.Monitors[i].IngestURL = ingestURL
		c.Monitors[i].SignalFxAccessToken = c.SignalFxAccessToken
		c.Monitors[i].Hostname = c.Hostname
		c.Monitors[i].Filter = filterSet
		c.Monitors[i].ProcFSPath = c.ProcFSPath
		// Top level interval serves as a default
		c.Monitors[i].IntervalSeconds = utils.FirstNonZero(c.Monitors[i].IntervalSeconds, c.IntervalSeconds)
		c.Monitors[i].MetaStore = metaStore
		c.Monitors[i].InternalMetricsSocketPath = c.InternalMetricsSocketPath
	}

	for i := range c.Observers {
		c.Observers[i].Hostname = c.Hostname
	}

	c.Writer.IngestURL = ingestURL
	c.Writer.APIURL = apiURL
	c.Writer.Filter = filterSet
	c.Writer.SignalFxAccessToken = c.SignalFxAccessToken
	c.Writer.GlobalDimensions = c.GlobalDimensions
}

// LogConfig contains configuration related to logging
type LogConfig struct {
	Level string `yaml:"level,omitempty" default:"info"`
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

// CustomConfigurable should be implemented by config structs that have the
// concept of generic other config that is initially deserialized into a
// map[string]interface{} to be later transformed to another form.
type CustomConfigurable interface {
	ExtraConfig() map[string]interface{}
}

var validCollectdLogLevels = set.NewNonTS("debug", "info", "notice", "warning", "err")

// CollectdConfig high-level configurations
type CollectdConfig struct {
	DisableCollectd      bool   `yaml:"disableCollectd,omitempty" default:"false"`
	Timeout              int    `yaml:"timeout,omitempty" default:"40"`
	ReadThreads          int    `yaml:"readThreads,omitempty" default:"5"`
	WriteQueueLimitHigh  int    `yaml:"writeQueueLimitHigh,omitempty" default:"500000"`
	WriteQueueLimitLow   int    `yaml:"writeQueueLimitLow,omitempty" default:"400000"`
	CollectInternalStats bool   `yaml:"collectInternalStats,omitempty" default:"true"`
	LogLevel             string `yaml:"logLevel,omitempty" default:"notice"`
	IntervalSeconds      int    `yaml:"intervalSeconds" default:"0"`
	WriteServerIPAddr    string `yaml:"writeServerIPAddr" default:"127.9.8.7"`
	WriteServerPort      uint16 `yaml:"writeServerPort" default:"14839"`
	// The following are propagated from the top-level config
	HostFSPath           string             `yaml:"-"`
	Hostname             string             `yaml:"-"`
	Filter               *filters.FilterSet `yaml:"-"`
	SignalFxAccessToken  string             `yaml:"-"`
	IngestURL            string             `yaml:"-"`
	GlobalDimensions     map[string]string  `yaml:"-"`
	HasGenericJMXMonitor bool               `yaml:"-"`
}

// Validate the collectd specific config
func (cc *CollectdConfig) Validate() error {
	if !validCollectdLogLevels.Has(cc.LogLevel) {
		return errors.Errorf("Invalid collectd log level %s.  Valid choices are %v",
			cc.LogLevel, validCollectdLogLevels)
	}

	return nil
}

// WriteServerURL is the local address served by the agent where collect should
// write datapoints
func (cc *CollectdConfig) WriteServerURL() string {
	return fmt.Sprintf("http://%s:%d/", cc.WriteServerIPAddr, cc.WriteServerPort)
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
