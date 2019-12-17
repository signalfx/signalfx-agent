package elasticsearch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/elasticsearch/client"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// Config for this monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host                 string `yaml:"host" validate:"required"`
	Port                 string `yaml:"port" validate:"required"`
	// Username used to access Elasticsearch stats API
	Username string `yaml:"username"`
	// Password used to access Elasticsearch stats API
	Password string `yaml:"password" neverLog:"true"`
	// Whether to use https or not
	UseHTTPS bool `yaml:"useHTTPS"`
	// Whether to skip TLS certificate validation.
	SkipVerify bool `yaml:"skipVerify"`
	// Cluster name to which the node belongs. This is an optional config that
	// will override the cluster name fetched from a node and will be used to
	// populate the plugin_instance dimension
	Cluster string `yaml:"cluster"`
	// Enable Index stats. If set to true, by default the a subset of index
	// stats will be collected (see docs for list of default index metrics collected).
	EnableIndexStats *bool `yaml:"enableIndexStats" default:"true"`
	// Indexes to collect stats from (by default stats from all indexes are collected)
	Indexes []string `yaml:"indexes"`
	// Interval to report IndexStats on
	IndexStatsIntervalSeconds int `yaml:"indexStatsIntervalSeconds" default:"60"`
	// Collect only aggregated index stats across all indexes
	IndexSummaryOnly bool `yaml:"indexSummaryOnly"`
	// Collect index stats only from Master node
	IndexStatsMasterOnly *bool `yaml:"indexStatsMasterOnly" default:"true"`
	// EnableClusterHealth enables reporting on the cluster health
	EnableClusterHealth *bool `yaml:"enableClusterHealth" default:"true"`
	// Whether or not non master nodes should report cluster health
	ClusterHealthStatsMasterOnly *bool `yaml:"clusterHealthStatsMasterOnly" default:"true"`
	// Enable enhanced HTTP stats
	EnableEnhancedHTTPStats bool `yaml:"enableEnhancedHTTPStats"`
	// Enable enhanced JVM stats
	EnableEnhancedJVMStats bool `yaml:"enableEnhancedJVMStats"`
	// Enable enhanced Process stats
	EnableEnhancedProcessStats bool `yaml:"enableEnhancedProcessStats"`
	// Enable enhanced ThreadPool stats
	EnableEnhancedThreadPoolStats bool `yaml:"enableEnhancedThreadPoolStats"`
	// Enable enhanced Transport stats
	EnableEnhancedTransportStats bool `yaml:"enableEnhancedTransportStats"`
	// Enable enhanced node level index stats groups. A list of index stats
	// groups for which to collect enhanced stats
	EnableEnhancedNodeStatsForIndexGroups []string `yaml:"enableEnhancedNodeIndicesStats"`
	// ThreadPools to report threadpool node stats on
	ThreadPools []string `yaml:"threadPools" default:"[\"search\", \"index\"]"`
	// Enable Cluster level stats. These stats report only from master Elasticserach nodes
	EnableEnhancedClusterHealthStats bool `yaml:"enableEnhancedClusterHealthStats"`
	// Enable enhanced index level index stats groups. A list of index stats groups
	// for which to collect enhanced stats
	EnableEnhancedIndexStatsForIndexGroups []string `yaml:"enableEnhancedIndexStatsForIndexGroups"`
	// To enable index stats from only primary shards. By default the index stats collected
	// are aggregated across all shards
	EnableIndexStatsPrimaries bool `yaml:"enableIndexStatsPrimaries"`
	// How often to refresh metadata about the node and cluster
	MetadataRefreshIntervalSeconds int `yaml:"metadataRefreshIntervalSeconds" default:"30"`
}

// Monitor for conviva metrics
type Monitor struct {
	Output types.FilteringOutput
	cancel context.CancelFunc
	ctx    context.Context
	logger *utils.ThrottledLogger
}

func init() {
	monitors.Register(&client.MonitorMetadata, func() interface{} { return &Monitor{} }, &Config{})
}

type sharedInfo struct {
	nodeIsCurrentMaster  bool
	defaultDimensions    map[string]string
	nodeMetricDimensions map[string]string
	lock                 sync.RWMutex
	logger               *utils.ThrottledLogger
}

func (sinfo *sharedInfo) fetchNodeAndClusterMetadata(esClient client.ESHttpClient, configuredClusterName string) error {
	// Collect info about master for the cluster
	masterInfoOutput, err := esClient.GetMasterNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to GET master node info: %v", err)
	}

	// Collect node info
	nodeInfoOutput, err := esClient.GetNodeInfo()
	if err != nil {
		return fmt.Errorf("failed to GET node info: %v", err)
	}

	// Hold the lock while updating info shared between different fetchers
	sinfo.lock.Lock()
	defer sinfo.lock.Unlock()

	nodeDimensions, err := prepareNodeMetricsDimensions(nodeInfoOutput.Nodes)
	if err != nil {
		return fmt.Errorf("failed to prepare node dimensions: %v", err)
	}

	sinfo.nodeIsCurrentMaster, err = isCurrentMaster(nodeDimensions["node_id"], masterInfoOutput)
	if err != nil {
		sinfo.logger.ThrottledWarning(err.Error())
		sinfo.nodeIsCurrentMaster = false
	}

	clusterName := nodeInfoOutput.ClusterName
	sinfo.defaultDimensions, err = prepareDefaultDimensions(configuredClusterName, clusterName)

	if err != nil {
		return fmt.Errorf("failed to prepare plugin_instance dimension: %v", err)
	}

	sinfo.nodeMetricDimensions = utils.MergeStringMaps(sinfo.defaultDimensions, nodeDimensions)

	return nil
}

// Returns all fields of a shared info object
func (sinfo *sharedInfo) getAllSharedInfo() (map[string]string, map[string]string, bool) {
	sinfo.lock.RLock()
	defer sinfo.lock.RUnlock()

	return sinfo.defaultDimensions, sinfo.nodeMetricDimensions, sinfo.nodeIsCurrentMaster
}

// Configure monitor
func (m *Monitor) Configure(c *Config) error {
	m.logger = utils.NewThrottledLogger(log.WithFields(log.Fields{"monitorType": client.MonitorType}), 20*time.Second)

	// conf is a config shallow copy that will be mutated and used to configure monitor
	conf := &Config{}
	*conf = *c
	// Setting metric group flags in conf for configured extra metrics
	if m.Output.HasEnabledMetricInGroup(client.GroupCluster) {
		conf.EnableEnhancedClusterHealthStats = true
	}
	if m.Output.HasEnabledMetricInGroup(client.GroupNodeHTTP) {
		conf.EnableEnhancedHTTPStats = true
	}
	if m.Output.HasEnabledMetricInGroup(client.GroupNodeJvm) {
		conf.EnableEnhancedJVMStats = true
	}
	if m.Output.HasEnabledMetricInGroup(client.GroupNodeProcess) {
		conf.EnableEnhancedProcessStats = true
	}
	if m.Output.HasEnabledMetricInGroup(client.GroupNodeThreadPool) {
		conf.EnableEnhancedThreadPoolStats = true
	}
	if m.Output.HasEnabledMetricInGroup(client.GroupNodeTransport) {
		conf.EnableEnhancedTransportStats = true
	}

	esClient := client.NewESClient(conf.Host, conf.Port, conf.UseHTTPS, conf.SkipVerify, conf.Username, conf.Password)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	var isInitialized bool

	// To handle metadata about metrics that is shared
	shared := sharedInfo{
		lock:   sync.RWMutex{},
		logger: m.logger,
	}

	utils.RunOnInterval(m.ctx, func() {
		// Fetch metadata from Elasticsearch nodes
		err := shared.fetchNodeAndClusterMetadata(esClient, conf.Cluster)

		// For any reason the monitor is not able to fetch cluster and node information upfront we capture that
		// to ensure that the stats are not sent in with faulty dimensions. The monitor will try to fetch this
		// information again every MetadataRefreshInterval seconds
		if err != nil {
			m.logger.WithError(err).Errorf(fmt.Sprintf("Failed to get Cluster and node metadata upfront. Will try again in %d s.", conf.MetadataRefreshIntervalSeconds))
			return
		}

		// The first time fetchNodeAndClusterMetadata succeeded the monitor can schedule fetchers for all the other stats
		if !isInitialized {
			// Collect Node level stats
			utils.RunOnInterval(m.ctx, func() {
				_, defaultNodeMetricDimensions, _ := shared.getAllSharedInfo()

				m.fetchNodeStats(esClient, conf, defaultNodeMetricDimensions)
			}, time.Duration(conf.IntervalSeconds)*time.Second)

			// Collect cluster level stats
			utils.RunOnInterval(m.ctx, func() {
				defaultDimensions, _, nodeIsCurrentMaster := shared.getAllSharedInfo()

				if !*conf.ClusterHealthStatsMasterOnly || nodeIsCurrentMaster {
					m.fetchClusterStats(esClient, conf, defaultDimensions)
				}
			}, time.Duration(conf.IntervalSeconds)*time.Second)

			// Collect Index stats
			if *conf.EnableIndexStats {
				utils.RunOnInterval(m.ctx, func() {
					defaultDimensions, _, nodeIsCurrentMaster := shared.getAllSharedInfo()

					// if "IndexStatsMasterOnly" is true collect stats only from master,
					// otherwise collect index stats from all nodes
					if !*conf.IndexStatsMasterOnly || nodeIsCurrentMaster {
						m.fetchIndexStats(esClient, conf, defaultDimensions)
					}
				}, time.Duration(conf.IndexStatsIntervalSeconds)*time.Second)
			}
			isInitialized = true
		}
	}, time.Duration(conf.MetadataRefreshIntervalSeconds)*time.Second)

	return nil
}

func (m *Monitor) fetchNodeStats(esClient client.ESHttpClient, conf *Config, defaultNodeDimensions map[string]string) {
	nodeStatsOutput, err := esClient.GetNodeAndThreadPoolStats()
	if err != nil {
		m.logger.WithError(err).Errorf("Failed to GET node stats")
		return
	}

	m.sendDatapoints(client.GetNodeStatsDatapoints(nodeStatsOutput, defaultNodeDimensions, utils.StringSliceToMap(conf.ThreadPools), utils.StringSliceToMap(conf.EnableEnhancedNodeStatsForIndexGroups), map[string]bool{
		client.HTTPStatsGroup:       conf.EnableEnhancedHTTPStats,
		client.JVMStatsGroup:        conf.EnableEnhancedJVMStats,
		client.ProcessStatsGroup:    conf.EnableEnhancedProcessStats,
		client.ThreadpoolStatsGroup: conf.EnableEnhancedThreadPoolStats,
		client.TransportStatsGroup:  conf.EnableEnhancedTransportStats,
	}))
}

func (m *Monitor) fetchClusterStats(esClient client.ESHttpClient, conf *Config, defaultDimensions map[string]string) {
	clusterStatsOutput, err := esClient.GetClusterStats()

	if err != nil {
		m.logger.WithError(err).Errorf("Failed to GET cluster stats")
		return
	}

	m.sendDatapoints(client.GetClusterStatsDatapoints(clusterStatsOutput, defaultDimensions, conf.EnableEnhancedClusterHealthStats))
}

func (m *Monitor) fetchIndexStats(esClient client.ESHttpClient, conf *Config, defaultDimensions map[string]string) {
	indexStatsOutput, err := esClient.GetIndexStats()

	if err != nil {
		m.logger.WithError(err).Errorf("Failed to GET index stats")
		return
	}

	if conf.IndexSummaryOnly {
		m.sendDatapoints(client.GetIndexStatsSummaryDatapoints(indexStatsOutput.AllIndexStats, defaultDimensions, utils.StringSliceToMap(conf.EnableEnhancedIndexStatsForIndexGroups), conf.EnableIndexStatsPrimaries))
		return
	}

	m.sendDatapoints(client.GetIndexStatsDatapoints(indexStatsOutput.Indices, utils.StringSliceToMap(conf.Indexes), defaultDimensions, utils.StringSliceToMap(conf.EnableEnhancedIndexStatsForIndexGroups), conf.EnableIndexStatsPrimaries))
}

// Prepares dimensions that are common to all datapoints from the monitor
func prepareDefaultDimensions(userProvidedClusterName string, queriedClusterName *string) (map[string]string, error) {
	dims := map[string]string{}
	clusterName := userProvidedClusterName

	if clusterName == "" {
		if queriedClusterName == nil {
			return nil, errors.New("failed to GET cluster name from Elasticsearch API")
		}
		clusterName = *queriedClusterName
	}

	// "plugin_instance" dimension is added to maintain backwards compatibility with built-in content
	dims["plugin_instance"] = clusterName
	dims["cluster"] = clusterName
	dims["plugin"] = client.MonitorType

	return dims, nil
}

func isCurrentMaster(nodeID string, masterInfoOutput *client.MasterInfoOutput) (bool, error) {
	masterNode := masterInfoOutput.MasterNode

	if masterNode == nil {
		return false, errors.New("unable to identify Elasticsearch cluster master node, assuming current node is not the current master")
	}

	return nodeID == *masterNode, nil
}

func prepareNodeMetricsDimensions(nodeInfo map[string]client.NodeInfo) (map[string]string, error) {
	var nodeID string

	if len(nodeInfo) != 1 {
		return nil, fmt.Errorf("expected info about exactly one node, received a map with %d entries", len(nodeInfo))
	}

	// nodes will have exactly one entry, for the current node since the monitor hits "_nodes/_local" endpoint
	for node := range nodeInfo {
		nodeID = node
	}

	if nodeID == "" {
		return nil, errors.New("failed to obtain Elasticsearch node id")
	}

	dims := map[string]string{}
	dims["node_id"] = nodeID

	nodeName := nodeInfo[nodeID].Name

	if nodeName != nil {
		dims["node_name"] = *nodeName
	}

	return dims, nil
}

func (m *Monitor) sendDatapoints(dps []*datapoint.Datapoint) {
	// This is the filtering in place trick from https://github.com/golang/go/wiki/SliceTricks#filter-in-place
	n := 0
	for i := range dps {
		if dps[i] == nil {
			continue
		}
		dps[n] = dps[i]
		n++
	}
	m.Output.SendDatapoints(dps[:n]...)
}

// GetExtraMetrics returns additional metrics to allow through.
func (c *Config) GetExtraMetrics() []string {
	var extraMetrics []string
	if c.EnableEnhancedClusterHealthStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupCluster]...)
	}
	if c.EnableEnhancedHTTPStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupNodeHTTP]...)
	}
	if c.EnableEnhancedJVMStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupNodeJvm]...)
	}
	if c.EnableEnhancedProcessStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupNodeProcess]...)
	}
	if c.EnableEnhancedThreadPoolStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupNodeThreadPool]...)
	}
	if c.EnableEnhancedTransportStats {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupNodeTransport]...)
	}
	enhancedStatsForIndexGroups := utils.StringSliceToMap(append(c.EnableEnhancedNodeStatsForIndexGroups, c.EnableEnhancedIndexStatsForIndexGroups...))
	if enhancedStatsForIndexGroups[client.StoreStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesStore]...)
	}
	if enhancedStatsForIndexGroups[client.IndexingStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesIndexing]...)
	}
	if enhancedStatsForIndexGroups[client.GetStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesGet]...)
	}
	if enhancedStatsForIndexGroups[client.SearchStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesSearch]...)
	}
	if enhancedStatsForIndexGroups[client.MergesStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesMerges]...)
	}
	if enhancedStatsForIndexGroups[client.RefreshStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesRefresh]...)
	}
	if enhancedStatsForIndexGroups[client.FlushStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesFlush]...)
	}
	if enhancedStatsForIndexGroups[client.WarmerStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesWarmer]...)
	}
	if enhancedStatsForIndexGroups[client.QueryCacheStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesQueryCache]...)
	}
	if enhancedStatsForIndexGroups[client.FilterCacheStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesFilterCache]...)
	}
	if enhancedStatsForIndexGroups[client.FieldDataStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesFielddata]...)
	}
	if enhancedStatsForIndexGroups[client.CompletionStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesCompletion]...)
	}
	if enhancedStatsForIndexGroups[client.TranslogStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesTranslog]...)
	}
	if enhancedStatsForIndexGroups[client.RequestCacheStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesRequestCache]...)
	}
	if enhancedStatsForIndexGroups[client.RecoveryStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesRecovery]...)
	}
	if enhancedStatsForIndexGroups[client.IDCacheStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesIDCache]...)
	}
	if enhancedStatsForIndexGroups[client.SuggestStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesSuggest]...)
	}
	if enhancedStatsForIndexGroups[client.PercolateStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesPercolate]...)
	}
	if enhancedStatsForIndexGroups[client.SegmentsStatsGroup] {
		extraMetrics = append(extraMetrics, client.GroupMetricsMap[client.GroupIndicesSegments]...)
	}
	return extraMetrics
}

// Shutdown stops the metric sync
func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
