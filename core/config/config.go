package config

import (
	"fmt"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	fqdn "github.com/ShowMax/go-fqdn"
	"github.com/signalfx/neo-agent/core/filters"
	"github.com/signalfx/neo-agent/utils"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	// TODO: Consider whether to allow store configuration from the main config
	// file.  There is a major chicken/egg problem with this and reloading
	// stores is very tricky.  Right now, stores can only be configured via
	// envvars and I think it is best to keep it that way.
	//Stores              map[string]StoreConfig `yaml:"stores,omitempty" default:"{}"`
	SignalFxAccessToken string `yaml:"signalFxAccessToken,omitempty"`
	// The ingest URL for SignalFx, without the path
	IngestURL string `yaml:"ingestUrl,omitempty" default:"https://ingest.signalfx.com"`
	// The hostname that will be reported as the "host" dimension on metrics
	// for which host applies
	Hostname string `yaml:"hostname,omitempty"`
	// How often to send metrics to SignalFx.  Monitors can't override this
	// individually.
	IntervalSeconds int `yaml:"intervalSeconds,omitempty" default:"10"`
	// Dimensions that will be automatically added to all metrics reported
	GlobalDimensions map[string]string `yaml:"globalDimensions,omitempty" default:"{}"`
	Observers        []ObserverConfig  `yaml:"observers,omitempty" default:"[]"`
	Monitors         []MonitorConfig   `yaml:"monitors,omitempty" default:"[]"`
	Logging          LogConfig         `yaml:"logging,omitempty" default:"{}"`
	// Configure the underlying collectd daemon
	Collectd         CollectdConfig `yaml:"collectd,omitempty" default:"{}"`
	MetricsToExclude []MetricFilter `yaml:"metricsToExclude,omitempty" default:"[]"`
	ProcFSPath       string         `yaml:"procFSPath,omitempty" default:"/proc"`
}

func (c *Config) setDefaultHostname() {
	fqdn := fqdn.Get()
	if fqdn == "unknown" {
		log.Info("Error getting fully qualified hostname")
	} else {
		log.Infof("Using hostname %s", fqdn)
		c.Hostname = fqdn
	}
}

func (c *Config) Initialize() (*Config, error) {
	c.overrideFromEnv()

	c.setDefaultHostname()

	if !c.validate() {
		return nil, fmt.Errorf("Configuration did not validate!")
	}

	c.propagateValuesDown()
	for i := range c.Monitors {
		c.Monitors[i].EnsureID()
	}

	return c, nil
}

func (c *Config) IngestURLAsURL() *url.URL {
	if url, err := url.Parse(c.IngestURL); err == nil {
		return url
	}
	return nil
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

// Validate everything except for Observers and Monitors
func (c *Config) validate() bool {
	valid := true

	if c.SignalFxAccessToken == "" {
		log.Error("signalFxAccessToken must be set!")
		valid = false
	}
	if _, err := url.Parse(c.IngestURL); err != nil {
		log.WithFields(log.Fields{
			"ingestURL": c.IngestURL,
			"error":     err,
		}).Error("ingestURL is not a valid URL")
	}

	return valid
}

func (c *Config) makeFilterSet() *filters.FilterSet {
	fs := make([]*filters.Filter, 0)
	for _, mte := range c.MetricsToExclude {
		dims := mte.ConvertDimensionsMapForSliceValues()
		mte.ConvertMetricNameToSlice()
		fs = append(fs, filters.New(mte.MonitorType, mte.MetricNames, dims))
	}

	return &filters.FilterSet{
		Filters: fs,
	}
}

// Send values from the top of the config down to nested configs that might
// need them
func (c *Config) propagateValuesDown() {
	filterSet := c.makeFilterSet()
	for i := range c.Monitors {
		if url, err := url.Parse(c.IngestURL); err == nil {
			c.Monitors[i].IngestURL = url
		}
		c.Monitors[i].GlobalDimensions = c.GlobalDimensions
		c.Monitors[i].SignalFxAccessToken = c.SignalFxAccessToken
		c.Monitors[i].Hostname = c.Hostname
		c.Monitors[i].Filter = filterSet
		c.Monitors[i].ProcFSPath = c.ProcFSPath
		// Top level interval serves as a default
		c.Monitors[i].IntervalSeconds = utils.FirstNonZero(c.Monitors[i].IntervalSeconds, c.IntervalSeconds)
	}

	c.Collectd.Hostname = c.Hostname
	c.Collectd.IntervalSeconds = c.IntervalSeconds
	c.Collectd.Filter = filterSet
}

type LogConfig struct {
	Level string `yaml:"level,omitempty" default:"info"`
	// TODO: Support log file output and other log targets
}

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

type MonitorID string

type MonitorConfig struct {
	Type string `yaml:"type,omitempty"`
	// Id can be used to uniquely identify monitors so that they can be
	// reconfigured in place instead of destroyed and recreated
	Id              MonitorID         `yaml:"id,omitempty"`
	DiscoveryRule   string            `yaml:"discoveryRule,omitempty"`
	ExtraDimensions map[string]string `yaml:"extraDimensions,omitempty" default:"{}"`
	// K8s pod label keys to send as dimensions
	K8sLabelDimensions []string `yaml:"labelDimensions,omitempty" default:"[]"`
	// If unset or 0, will default to the top-level IntervalSeconds value
	IntervalSeconds int                    `yaml:"intervalSeconds,omitempty" default:"0"`
	OtherConfig     map[string]interface{} `yaml:",inline" default:"{}" json:"-"`
	// The remaining are propagated from the top-level config and cannot be set
	// by the user directly on the monitor
	IngestURL           *url.URL           `yaml:"-"`
	SignalFxAccessToken string             `yaml:"-"`
	Hostname            string             `yaml:"-"`
	Filter              *filters.FilterSet `yaml:"-"`
	// Most monitors can ignore this
	GlobalDimensions map[string]string `yaml:"-" default:"{}"`
	ProcFSPath       string            `yaml:"-"`
}

func (mc *MonitorConfig) GetOtherConfig() map[string]interface{} {
	return mc.OtherConfig
}

func (mc *MonitorConfig) EnsureID() {
	if len(mc.Id) == 0 {
		mc.Id = MonitorID(fmt.Sprintf("%s-%d", mc.Type, getNextIdFor(mc.Type)))
	}
}

type ObserverConfig struct {
	Type string `yaml:"type,omitempty"`
	// Id can be used to uniquely identify observers so that they can be
	// reconfigured in place instead of destroyed and recreated
	Id          string                 `yaml:"id,omitempty"`
	OtherConfig map[string]interface{} `yaml:",inline" default:"{}"`
}

func (oc *ObserverConfig) GetOtherConfig() map[string]interface{} {
	return oc.OtherConfig
}

type CustomConfigurable interface {
	GetOtherConfig() map[string]interface{}
}

// Collectd high-level configurations
type CollectdConfig struct {
	DisableCollectd      bool               `yaml:"disableCollectd,omitempty" default:"false"`
	IntervalSeconds      int                `yaml:"intervalSeconds,omitempty" default:"10"`
	Timeout              int                `yaml:"timeout,omitempty" default:"40"`
	ReadThreads          int                `yaml:"readThreads,omitempty" default:"5"`
	WriteQueueLimitHigh  int                `yaml:"writeQueueLimitHigh,omitempty" default:"500000"`
	WriteQueueLimitLow   int                `yaml:"writeQueueLimitLow,omitempty" default:"400000"`
	CollectInternalStats bool               `yaml:"collectInternalStats,omitempty" default:"false"`
	LogLevel             string             `yaml:"logLevel,omitempty" default:"notice"`
	Hostname             string             `yaml:"-"`
	Filter               *filters.FilterSet `yaml:"-"`
}

// MetricFiltering describes a set of subtractive filters applied to datapoints
// right before they are sent.
type MetricFilter struct {
	// Can map to either a []string or simple string
	Dimensions  map[string]interface{} `default:"{}"`
	MetricNames []string               `default:"[]"`
	MetricName  string
	MonitorType string
}

func (mf *MetricFilter) ConvertDimensionsMapForSliceValues() map[string][]string {
	dims := make(map[string][]string)
	for k, d := range mf.Dimensions {
		if s, ok := d.(string); ok {
			dims[k] = []string{s}
		} else if interfaceSlice, ok := d.([]interface{}); ok {
			ss := utils.InterfaceSliceToStringSlice(interfaceSlice)
			if ss != nil {
				dims[k] = ss
			}
		}

		if dims[k] == nil {
			log.WithFields(log.Fields{
				"dimensionFilter": k,
				"value":           d,
				"type":            reflect.ValueOf(d).Type(),
			}).Error("Invalid dimension filter")
			return nil
		}
	}
	return dims
}

func (mf *MetricFilter) ConvertMetricNameToSlice() {
	if mf.MetricName != "" {
		mf.MetricNames = append(mf.MetricNames, mf.MetricName)
	}
}

type StoreConfig struct {
	OtherConfig map[string]interface{} `yaml:",inline,omitempty" default:"{}"`
}

func (sc *StoreConfig) GetOtherConfig() map[string]interface{} {
	return sc.OtherConfig
}

var (
	// EnvReplacer replaces . and - with _
	EnvReplacer   = strings.NewReplacer(".", "_", "-", "_")
	configTimeout = 10 * time.Second
)

var _ids = map[string]int{}

// Used to ensure unique IDs for monitors and observers
func getNextIdFor(name string) int {
	_ids[name] += 1
	return _ids[name]
}
