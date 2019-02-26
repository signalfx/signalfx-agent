package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	"github.com/signalfx/signalfx-agent/internal/utils"
	log "github.com/sirupsen/logrus"
)

const monitorType = "redis"

// MONITOR(redis): Monitors a Redis KV store instance.
//
// Sample YAML configuration:
//
// ```yaml
// monitors:
// - type: redis
//   host: 127.0.0.1
//   port: 9100
// ```
//
// Sample YAML configuration with list lengths:
//
// ```yaml
// monitors:
// - type: collectd/redis
//   host: 127.0.0.1
//   port: 9100
//   listLengths:
//   - databaseIndex: 0
//     keyPattern: 'mylist*'
// ```

var logger = log.WithFields(log.Fields{"monitorType": monitorType})

func init() {
	monitors.Register(monitorType, func() interface{} { return &Monitor{} }, &Config{})
}

// ListLength defines a database index and key pattern for sending list lengths
type ListLength struct {
	// The database index.
	DBIndex uint16 `yaml:"databaseIndex" validate:"required"`
	// Can be a globbed pattern (only * is supported), in which case all keys
	// matching that glob will be processed.  The pattern should be placed in
	// single quotes (').  Ex. `'mylist*'`
	KeyPattern string `yaml:"keyPattern" validate:"required"`
}

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host                 string `yaml:"host" validate:"required"`
	Port                 uint16 `yaml:"port" validate:"required"`
	// The name for the node is a canonical identifier which is used as plugin
	// instance. It is limited to 64 characters in length.  (**default**: "{host}:{port}")
	Name string `yaml:"name"`
	// Password to use for authentication.
	Auth string `yaml:"auth" neverLog:"true"`
	// Specify a pattern of keys to lists for which to send their length as a
	// metric.
	ListLengths []ListLength `yaml:"listLengths"`
	// A list of metrics to additionally include.  This is a list of strings,
	// the values of which should be the name of the metric as it appears in
	// the Redis INFO command output (i.e. everything before the `:`).  The
	// values from the INFO command must be numeric (i.e. the part after the
	// `:` in the INFO output).
	ExtraMetrics []string `yaml:"extraMetrics"`
}

// Monitor for prometheus exporter metrics
type Monitor struct {
	Output types.Output
	cancel func()
	client *redis.Client
}

// Configure the monitor and kick off volume metric syncing
func (m *Monitor) Configure(conf *Config) error {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())

	m.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Password: conf.Auth,
	})

	utils.RunOnInterval(ctx, func() {
		dps, err := fetchRedisMetrics(m.client, conf.ExtraMetrics)
		if err != nil {
			logger.WithError(err).Error("Could not get Redis metrics")
			return
		}

		for i := range dps {
			m.Output.SendDatapoint(dps[i])
		}
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

func fetchRedisMetrics(client *redis.Client, extraMetrics []string) ([]*datapoint.Datapoint, error) {
	infoStr, err := client.Info().Result()
	if err != nil {
		return nil, err
	}

	infoMap := parseInfoString(infoStr)
	return metricsFromData(infoMap, utils.StringSliceToMap(extraMetrics))
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
