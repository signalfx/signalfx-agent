// Code generated by monitor-code-gen. DO NOT EDIT.

package redis

import (
	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors"
)

const monitorType = "prometheus/redis"

var groupSet = map[string]bool{}

const (
	redisAofCurrentRewriteDurationSec      = "redis_aof_current_rewrite_duration_sec"
	redisAofEnabled                        = "redis_aof_enabled"
	redisAofLastRewriteDurationSec         = "redis_aof_last_rewrite_duration_sec"
	redisAofRewriteInProgress              = "redis_aof_rewrite_in_progress"
	redisAofRewriteScheduled               = "redis_aof_rewrite_scheduled"
	redisBlockedClients                    = "redis_blocked_clients"
	redisClientBiggestInputBuf             = "redis_client_biggest_input_buf"
	redisClientLongestOutputList           = "redis_client_longest_output_list"
	redisClusterEnabled                    = "redis_cluster_enabled"
	redisCommandCallDurationSecondsCount   = "redis_command_call_duration_seconds_count"
	redisCommandCallDurationSecondsSum     = "redis_command_call_duration_seconds_sum"
	redisCommandsProcessedTotal            = "redis_commands_processed_total"
	redisConfigMaxclients                  = "redis_config_maxclients"
	redisConfigMaxmemory                   = "redis_config_maxmemory"
	redisConnectedClients                  = "redis_connected_clients"
	redisConnectedSlaves                   = "redis_connected_slaves"
	redisConnectionsReceivedTotal          = "redis_connections_received_total"
	redisDbAvgTTLSeconds                   = "redis_db_avg_ttl_seconds"
	redisDbKeys                            = "redis_db_keys"
	redisDbKeysExpiring                    = "redis_db_keys_expiring"
	redisEvictedKeysTotal                  = "redis_evicted_keys_total"
	redisExpiredKeysTotal                  = "redis_expired_keys_total"
	redisExporterBuildInfo                 = "redis_exporter_build_info"
	redisExporterLastScrapeDurationSeconds = "redis_exporter_last_scrape_duration_seconds"
	redisExporterLastScrapeError           = "redis_exporter_last_scrape_error"
	redisExporterScrapesTotal              = "redis_exporter_scrapes_total"
	redisInstanceInfo                      = "redis_instance_info"
	redisInstantaneousInputKbps            = "redis_instantaneous_input_kbps"
	redisInstantaneousOpsPerSec            = "redis_instantaneous_ops_per_sec"
	redisInstantaneousOutputKbps           = "redis_instantaneous_output_kbps"
	redisKeyspaceHitsTotal                 = "redis_keyspace_hits_total"
	redisKeyspaceMissesTotal               = "redis_keyspace_misses_total"
	redisLatestForkUsec                    = "redis_latest_fork_usec"
	redisLoadingDumpFile                   = "redis_loading_dump_file"
	redisMasterReplOffset                  = "redis_master_repl_offset"
	redisMemoryFragmentationRatio          = "redis_memory_fragmentation_ratio"
	redisMemoryMaxBytes                    = "redis_memory_max_bytes"
	redisMemoryUsedBytes                   = "redis_memory_used_bytes"
	redisMemoryUsedLuaBytes                = "redis_memory_used_lua_bytes"
	redisMemoryUsedPeakBytes               = "redis_memory_used_peak_bytes"
	redisMemoryUsedRssBytes                = "redis_memory_used_rss_bytes"
	redisNetInputBytesTotal                = "redis_net_input_bytes_total"
	redisNetOutputBytesTotal               = "redis_net_output_bytes_total"
	redisProcessID                         = "redis_process_id"
	redisPubsubChannels                    = "redis_pubsub_channels"
	redisPubsubPatterns                    = "redis_pubsub_patterns"
	redisRdbChangesSinceLastSave           = "redis_rdb_changes_since_last_save"
	redisRdbCurrentBgsaveDurationSec       = "redis_rdb_current_bgsave_duration_sec"
	redisRdbLastBgsaveDurationSec          = "redis_rdb_last_bgsave_duration_sec"
	redisRejectedConnectionsTotal          = "redis_rejected_connections_total"
	redisReplicationBacklogBytes           = "redis_replication_backlog_bytes"
	redisSlowlogLength                     = "redis_slowlog_length"
	redisStartTimeSeconds                  = "redis_start_time_seconds"
	redisUp                                = "redis_up"
	redisUptimeInSeconds                   = "redis_uptime_in_seconds"
	redisUsedCPUSys                        = "redis_used_cpu_sys"
	redisUsedCPUSysChildren                = "redis_used_cpu_sys_children"
	redisUsedCPUUser                       = "redis_used_cpu_user"
	redisUsedCPUUserChildren               = "redis_used_cpu_user_children"
)

var metricSet = map[string]monitors.MetricInfo{
	redisAofCurrentRewriteDurationSec:      {Type: datapoint.Gauge},
	redisAofEnabled:                        {Type: datapoint.Gauge},
	redisAofLastRewriteDurationSec:         {Type: datapoint.Gauge},
	redisAofRewriteInProgress:              {Type: datapoint.Gauge},
	redisAofRewriteScheduled:               {Type: datapoint.Gauge},
	redisBlockedClients:                    {Type: datapoint.Gauge},
	redisClientBiggestInputBuf:             {Type: datapoint.Gauge},
	redisClientLongestOutputList:           {Type: datapoint.Gauge},
	redisClusterEnabled:                    {Type: datapoint.Gauge},
	redisCommandCallDurationSecondsCount:   {Type: datapoint.Gauge},
	redisCommandCallDurationSecondsSum:     {Type: datapoint.Gauge},
	redisCommandsProcessedTotal:            {Type: datapoint.Gauge},
	redisConfigMaxclients:                  {Type: datapoint.Gauge},
	redisConfigMaxmemory:                   {Type: datapoint.Gauge},
	redisConnectedClients:                  {Type: datapoint.Gauge},
	redisConnectedSlaves:                   {Type: datapoint.Gauge},
	redisConnectionsReceivedTotal:          {Type: datapoint.Gauge},
	redisDbAvgTTLSeconds:                   {Type: datapoint.Gauge},
	redisDbKeys:                            {Type: datapoint.Gauge},
	redisDbKeysExpiring:                    {Type: datapoint.Gauge},
	redisEvictedKeysTotal:                  {Type: datapoint.Gauge},
	redisExpiredKeysTotal:                  {Type: datapoint.Gauge},
	redisExporterBuildInfo:                 {Type: datapoint.Gauge},
	redisExporterLastScrapeDurationSeconds: {Type: datapoint.Gauge},
	redisExporterLastScrapeError:           {Type: datapoint.Gauge},
	redisExporterScrapesTotal:              {Type: datapoint.Gauge},
	redisInstanceInfo:                      {Type: datapoint.Gauge},
	redisInstantaneousInputKbps:            {Type: datapoint.Gauge},
	redisInstantaneousOpsPerSec:            {Type: datapoint.Gauge},
	redisInstantaneousOutputKbps:           {Type: datapoint.Gauge},
	redisKeyspaceHitsTotal:                 {Type: datapoint.Gauge},
	redisKeyspaceMissesTotal:               {Type: datapoint.Gauge},
	redisLatestForkUsec:                    {Type: datapoint.Gauge},
	redisLoadingDumpFile:                   {Type: datapoint.Gauge},
	redisMasterReplOffset:                  {Type: datapoint.Gauge},
	redisMemoryFragmentationRatio:          {Type: datapoint.Gauge},
	redisMemoryMaxBytes:                    {Type: datapoint.Gauge},
	redisMemoryUsedBytes:                   {Type: datapoint.Gauge},
	redisMemoryUsedLuaBytes:                {Type: datapoint.Gauge},
	redisMemoryUsedPeakBytes:               {Type: datapoint.Gauge},
	redisMemoryUsedRssBytes:                {Type: datapoint.Gauge},
	redisNetInputBytesTotal:                {Type: datapoint.Gauge},
	redisNetOutputBytesTotal:               {Type: datapoint.Gauge},
	redisProcessID:                         {Type: datapoint.Gauge},
	redisPubsubChannels:                    {Type: datapoint.Gauge},
	redisPubsubPatterns:                    {Type: datapoint.Gauge},
	redisRdbChangesSinceLastSave:           {Type: datapoint.Gauge},
	redisRdbCurrentBgsaveDurationSec:       {Type: datapoint.Gauge},
	redisRdbLastBgsaveDurationSec:          {Type: datapoint.Gauge},
	redisRejectedConnectionsTotal:          {Type: datapoint.Gauge},
	redisReplicationBacklogBytes:           {Type: datapoint.Gauge},
	redisSlowlogLength:                     {Type: datapoint.Gauge},
	redisStartTimeSeconds:                  {Type: datapoint.Gauge},
	redisUp:                                {Type: datapoint.Gauge},
	redisUptimeInSeconds:                   {Type: datapoint.Gauge},
	redisUsedCPUSys:                        {Type: datapoint.Gauge},
	redisUsedCPUSysChildren:                {Type: datapoint.Gauge},
	redisUsedCPUUser:                       {Type: datapoint.Gauge},
	redisUsedCPUUserChildren:               {Type: datapoint.Gauge},
}

var defaultMetrics = map[string]bool{}

var groupMetricsMap = map[string][]string{}

var monitorMetadata = monitors.Metadata{
	MonitorType:       "prometheus/redis",
	DefaultMetrics:    defaultMetrics,
	Metrics:           metricSet,
	MetricsExhaustive: false,
	Groups:            groupSet,
	GroupMetricsMap:   groupMetricsMap,
	SendAll:           true,
}
