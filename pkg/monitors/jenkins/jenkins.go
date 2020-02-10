package jenkins

import (
	"context"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/common/httpclient"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	jc "github.com/signalfx/signalfx-agent/pkg/monitors/jenkins/client"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	healthEndpoint  = "metrics/%s/healthcheck/"
	metricsEndpoint = "metrics/%s/metrics/"
	pingEndPoint    = "metrics/%s/ping/"
)

// Config for this monitor
type Config struct {
	config.MonitorConfig  `yaml:",inline" acceptsEndpoints:"true"`
	httpclient.HTTPConfig `yaml:",inline"`

	// Host of the jenkins instance
	Host string `yaml:"host" validate:"required"`
	// Port of the jenkins instance
	Port string `yaml:"port" validate:"required"`

	// Api key to authenticate to the jenkins metrics plugin
	MetricsKey string `yaml:"metricsKey" validate:"required"`

	// Dimension with a value that uniquely identifies the jenkins cluster.
	// Jenkins master has no unique identifier, this dimension will be used for that purpose.
	// In addition, the monitor tags the jenkins_cluster dimension with the instance labels.
	JenkinsCluster string `yaml:"jenkins_cluster" validate:"required"`
}

type Monitor struct {
	Output types.FilteringOutput
	cancel context.CancelFunc
	ctx    context.Context
	// JobsMetricsState map (key:job_name and value:JobMetricsState) used to keep track of builds state
	JobsMetricsState map[string]*JobMetricsState
}

var logger = utils.NewThrottledLogger(log.WithFields(log.Fields{"monitorType": monitorType}), 20*time.Second)

func init() {
	monitors.Register(&monitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

// Configure monitor
func (m *Monitor) Configure(conf *Config) error {
	httpClient, err := conf.HTTPConfig.Build()
	if err != nil {
		return err
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.JobsMetricsState = make(map[string]*JobMetricsState)

	uniqueDimMap := map[string]string{"jenkins_cluster": conf.JenkinsCluster}
	updateAndSendDPS := func(dps ...*datapoint.Datapoint) {
		for _, dp := range dps {
			dp.Dimensions = utils.MergeStringMaps(uniqueDimMap, dp.Dimensions)
		}
		m.Output.SendDatapoints(dps...)
	}
	// Although not currently used, but some gojenkins methods are not thread safe;
	// this anonymous func creates a jenkins client for each metrics collector.
	sendDPSPeriodically := func(fn func(jkClient jc.JenkinsClient)) {
		jkClient, err := jc.NewJenkinsClient(conf.Host, conf.MetricsKey, conf.Port, conf.Scheme(), httpClient)
		if err != nil {
			logger.WithError(err).Errorf("Failed to initialize the Jenkins Client")
			return
		}
		utils.RunOnInterval(m.ctx, func() { fn(jkClient) }, time.Duration(conf.IntervalSeconds)*time.Second)
	}

	metricsFuncs := []func(jc.JenkinsClient){
		func(jkClient jc.JenkinsClient) {
			dps := liveness(jkClient, conf.MetricsKey)
			updateAndSendDPS(dps...)
		},
		func(jkClient jc.JenkinsClient) {
			dps, err := nodeMetrics(jkClient, conf.MetricsKey)
			if err != nil {
				logger.WithError(err).Error("Could not get node metrics")
				return
			}
			updateAndSendDPS(dps...)
		},
		func(jkClient jc.JenkinsClient) {
			dps, err := healthMetrics(jkClient, conf.MetricsKey)
			if err != nil {
				logger.WithError(err).Error("Could not get health metrics")
				return
			}
			updateAndSendDPS(dps...)
		},
		func(jkClient jc.JenkinsClient) {
			dps, err := slaveStatus(jkClient)
			if err != nil {
				logger.WithError(err).Error("Could not get slave status")
				return
			}
			updateAndSendDPS(dps...)
		},
		func(jkClient jc.JenkinsClient) {
			dps, err := m.jobMetrics(jkClient)
			if err != nil {
				logger.WithError(err).Error("Could not get job metrics")
				return
			}
			updateAndSendDPS(dps...)
		},
	}

	for _, f := range metricsFuncs {
		sendDPSPeriodically(f)
	}

	jkClient, err := jc.NewJenkinsClient(conf.Host, conf.MetricsKey, conf.Port, conf.Scheme(), httpClient)
	if err != nil {
		return err
	}
	// Run unique dimension update
	utils.RunOnInterval(m.ctx, func() {
		dim, err := getUniqueDimension(jkClient, conf.JenkinsCluster)
		if err != nil {
			logger.WithError(err).Error("Could not get unique dimension")
			return
		}
		m.Output.SendDimensionUpdate(dim)
	}, time.Duration(conf.IntervalSeconds)*time.Second)

	return nil
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
