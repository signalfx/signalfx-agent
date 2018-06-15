package winperfcounters

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const monitorType = "telegraf/win_perf_counters"

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// Perfcounterobj represents a perfcounter object to monitor
type Perfcounterobj struct {
	ObjectName    string   `yaml:"objectName"`
	Counters      []string `yaml:"counters" default="[]"`
	Instances     []string `yaml:"instances" default="[]"`
	Measurement   string   `yaml:"measurement"`
	WarnOnMissing bool     `yaml:"warnOnMissing" default="false"`
	FailOnMissing bool     `yaml:"failOnMissing" default="false"`
	IncludeTotal  bool     `yaml:"includeTotal" default="false"`
}

// Config for this monitor
type Config struct {
	config.MonitorConfig `acceptsEndpoints:"false" deepcopier:"skip"`
	PrintValid           bool             `yaml:"PrintValid"`
	PreVistaSupport      bool             `yaml:"PreVistaSupport"`
	Object               []Perfcounterobj `yaml:"Objects" default:"[]"`
}

// Monitor for Utilization
type Monitor struct {
	Output  types.Output
	cancel  func()
	ctx     context.Context
	timeout time.Duration
}
