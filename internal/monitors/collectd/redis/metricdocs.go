package redis

// GAUGE(bytes.used_memory): Number of bytes allocated by Redis

// GAUGE(bytes.used_memory_lua): Number of bytes used by the Lua engine

// GAUGE(bytes.used_memory_peak): Peak Number of bytes allocated by Redis

// GAUGE(bytes.used_memory_rss): Number of bytes allocated by Redis as seen by the OS

// CUMULATIVE(counter.commands_processed): Total number of commands processed by the server

// CUMULATIVE(counter.connections_received): Total number of connections accepted by the server

// CUMULATIVE(counter.evicted_keys): Number of evicted keys due to maxmemory limit

// CUMULATIVE(counter.expired_keys): Total number of key expiration events

// CUMULATIVE(counter.lru_clock): Clock incrementing every minute, for LRU management

// CUMULATIVE(counter.rejected_connections): Number of connections rejected because of maxclients limit

// CUMULATIVE(counter.total_net_input_bytes): Total number of bytes inputted

// CUMULATIVE(counter.total_net_output_bytes): Total number of bytes outputted

// CUMULATIVE(counter.used_cpu_sys): System CPU consumed by the Redis server

// CUMULATIVE(counter.used_cpu_sys_children): System CPU consumed by the background processes

// CUMULATIVE(counter.used_cpu_user): User CPU consumed by the Redis server

// CUMULATIVE(counter.used_cpu_user_children): User CPU consumed by the background processes

// CUMULATIVE(derive.keyspace_hits): Number of successful lookup of keys in the main dictionary

// CUMULATIVE(derive.keyspace_misses): Number of failed lookup of keys in the main dictionary

// GAUGE(gauge.blocked_clients): Number of clients pending on a blocking call

// GAUGE(gauge.changes_since_last_save): Number of changes since the last dump

// GAUGE(gauge.client_biggest_input_buf): Biggest input buffer among current client connections

// GAUGE(gauge.client_longest_output_list): Longest output list among current client connections

// GAUGE(gauge.connected_clients): Number of client connections (excluding connections from slaves)

// GAUGE(gauge.connected_slaves): Number of connected slaves

// GAUGE(gauge.db0_avg_ttl): The average time to live for all keys in redis

// GAUGE(gauge.db0_expires): The total number of keys in redis that will expire

// GAUGE(gauge.db0_keys): The total number of keys stored in redis

// GAUGE(gauge.instantaneous_ops_per_sec): Number of commands processed per second

// GAUGE(gauge.key_llen): Length of an list key

// GAUGE(gauge.latest_fork_usec): Duration of the latest fork operation in microseconds

// GAUGE(gauge.master_last_io_seconds_ago): Number of seconds since the last interaction with master

// GAUGE(gauge.master_repl_offset): Master replication offset

// GAUGE(gauge.mem_fragmentation_ratio): Ratio between used_memory_rss and used_memory

// GAUGE(gauge.rdb_bgsave_in_progress): Flag indicating a RDB save is on-going

// GAUGE(gauge.repl_backlog_first_byte_offset): Slave replication backlog offset

// GAUGE(gauge.slave_repl_offset): Slave replication offset

// GAUGE(gauge.uptime_in_days): Number of days up

// GAUGE(gauge.uptime_in_seconds): Number of seconds up
