package redis

// All known metrics that should be sent as cumulative counters.  Everything
// else is assumed to be a gauge.
var cumulativeMetrics = map[string]bool{
	"commands_processed":     true,
	"connections_received":   true,
	"evicted_keys":           true,
	"expired_keys":           true,
	"lru_clock":              true,
	"rejected_connections":   true,
	"total_net_input_bytes":  true,
	"total_net_output_bytes": true,
	"used_cpu_sys":           true,
	"used_cpu_sys_children":  true,
	"keyspace_hits":          true,
	"keyspace_misses":        true,
}

// The metrics that are part of the built-in content and don't count as custom.
var nonCustomMetrics = map[string]bool{
	"redis.used_memory":      true,
	"redis.used_memory_rss":  true,
	"commands_processed":     true,
	"evicted_keys":           true,
	"expired_keys":           true,
	"rejected_connections":   true,
	"total_net_input_bytes":  true,
	"total_net_output_bytes": true,
	"used_cpu_sys":           true,
	"used_cpu_user":          true,
	"keyspace_hits":          true,
	"keyspace_misses":        true,
	"blocked_clients":        true,
	"connected_clients":      true,
	"master_repl_offset":     true,
	"slave_repl_offset":      true,
}

// GAUGE(redis.used_memory): Number of bytes allocated by Redis

// GAUGE(redis.used_memory_lua): Number of bytes used by the Lua engine

// GAUGE(redis.used_memory_peak): Peak Number of bytes allocated by Redis

// GAUGE(redis.used_memory_rss): Number of bytes allocated by Redis as seen by the OS

// CUMULATIVE(redis.commands_processed): Total number of commands processed by the server

// CUMULATIVE(redis.connections_received): Total number of connections accepted by the server

// CUMULATIVE(redis.evicted_keys): Number of evicted keys due to maxmemory limit

// CUMULATIVE(redis.expired_keys): Total number of key expiration events

// CUMULATIVE(redis.lru_clock): Clock incrementing every minute, for LRU management

// CUMULATIVE(redis.rejected_connections): Number of connections rejected because of maxclients limit

// CUMULATIVE(redis.total_net_input_bytes): Total number of bytes inputted

// CUMULATIVE(redis.total_net_output_bytes): Total number of bytes outputted

// CUMULATIVE(redis.used_cpu_sys): System CPU consumed by the Redis server

// CUMULATIVE(redis.used_cpu_sys_children): System CPU consumed by the background processes

// CUMULATIVE(redis.used_cpu_user): User CPU consumed by the Redis server

// CUMULATIVE(redis.used_cpu_user_children): User CPU consumed by the background processes

// CUMULATIVE(redis.keyspace_hits): Number of successful lookup of keys in the main dictionary

// CUMULATIVE(redis.keyspace_misses): Number of failed lookup of keys in the main dictionary

// GAUGE(redis.blocked_clients): Number of clients pending on a blocking call

// GAUGE(redis.changes_since_last_save): Number of changes since the last dump

// GAUGE(redis.client_biggest_input_buf): Biggest input buffer among current client connections

// GAUGE(redis.client_longest_output_list): Longest output list among current client connections

// GAUGE(redis.connected_clients): Number of client connections (excluding connections from slaves)

// GAUGE(redis.connected_slaves): Number of connected slaves

// GAUGE(redis.db<num>_avg_ttl): The average time to live for all keys in redis

// GAUGE(redis.db<num>_expires): The total number of keys in redis that will expire

// GAUGE(redis.db<num>_keys): The total number of keys stored in redis

// GAUGE(redis.instantaneous_ops_per_sec): Number of commands processed per second

// GAUGE(redis.key_llen): Length of an list key

// GAUGE(redis.latest_fork_usec): Duration of the latest fork operation in microseconds

// GAUGE(redis.master_last_io_seconds_ago): Number of seconds since the last interaction with master

// GAUGE(redis.master_repl_offset): Master replication offset

// GAUGE(redis.mem_fragmentation_ratio): Ratio between used_memory_rss and used_memory

// GAUGE(redis.rdb_bgsave_in_progress): Flag indicating a RDB save is on-going

// GAUGE(redis.repl_backlog_first_byte_offset): Slave replication backlog offset

// GAUGE(redis.slave_repl_offset): Slave replication offset

// GAUGE(redis.uptime_in_days): Number of days up

// GAUGE(redis.uptime_in_seconds): Number of seconds up
