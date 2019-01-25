package mssqlserver

// GAUGE(sqlserver_database_io.read_bytes): Bytes read by the database.
// GAUGE(sqlserver_database_io.read_latency_ms): Latency in milliseconds reading from the database.
// GAUGE(sqlserver_database_io.reads): Number of reads from the database.
// GAUGE(sqlserver_database_io.write_bytes): Bytes written to the database.
// GAUGE(sqlserver_database_io.write_latency_ms): Latency in milliseconds writing to the database.
// GAUGE(sqlserver_database_io.writes): Number of writes to the database.
// GAUGE(sqlserver_memory_clerks.size_kb.bound_trees): Size in KB of bound trees memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.buffer_pool): Size in KB of buffer pool memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.connection_pool): Size in KB of connection pool memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.general): Size in KB of general memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.in-memory_oltp): Size in KB of in in-memoory oltp memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.log_pool): Size in KB of log pool memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.memoryclerk_sqltrace): Size in KB of sql trace memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.schema_manager_user_store): Size in KB of user store schema manager memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.sos_node): Size in KB of sos node memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.sql_optimizer): Size in KB of SQL optimizer memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.sql_plans): Size in KB of sql plans memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.sql_reservations): Size in KB of sql reservations memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.sql_storage_engine): Size in KB of sql storage engine memory clerk.
// GAUGE(sqlserver_memory_clerks.size_kb.system_rowset_store): Size in KB of system rowset store memory clerk.
// GAUGE(sqlserver_performance.active_memory_grant_amount_kb): Amount of active memory in KB granted.
// GAUGE(sqlserver_performance.active_temp_tables): Number of active temporary tables.
// GAUGE(sqlserver_performance.background_writer_pages_persec): Rate per second of pages written in the background.
// GAUGE(sqlserver_performance.backup/restore_throughput_persec): Rate per second of backup/restore throughput.
// GAUGE(sqlserver_performance.batch_requests_persec): Rate per second of batch requests.
// GAUGE(sqlserver_performance.blocked_tasks): Number of blocked tasks.
// GAUGE(sqlserver_performance.buffer_cache_hit_ratio): Buffer cache hit ration.
// GAUGE(sqlserver_performance.bytes_received_from_replica_persec): Rate per second of bytes received from replicas.
// GAUGE(sqlserver_performance.bytes_sent_to_replica_persec): Rate per second of bytes sent to replicas.
// GAUGE(sqlserver_performance.bytes_sent_to_transport_persec): Rate per second of bytes sent to transports.
// GAUGE(sqlserver_performance.checkpoint_pages_persec): Rate per second of checkpoint pages.
// GAUGE(sqlserver_performance.cpu_limit_violation_count): Number of cpu limit violations.
// GAUGE(sqlserver_performance.cpu_usage_time): CPU usage time.
// GAUGE(sqlserver_performance.cpu_usage_percent): CPU usage percentage.
// GAUGE(sqlserver_performance.data_files_size_kb): Size in KB of data files.
// GAUGE(sqlserver_performance.disk_read_bytes_persec): Rate per second of bytes from disk.
// GAUGE(sqlserver_performance.disk_read_io_persec): Rate per second of read operations from disk.
// GAUGE(sqlserver_performance.disk_read_io_throttled_persec): Rate per second of throttled read operations.
// GAUGE(sqlserver_performance.disk_write_bytes_persec): Rate per second of bytes written to disk.
// GAUGE(sqlserver_performance.disk_write_io_persec): Rate per second of write operations to disk.
// GAUGE(sqlserver_performance.disk_write_io_throttled_persec): Rate per second of write operations throttled.
// GAUGE(sqlserver_performance.errors_persec): Rate of errors per second.
// GAUGE(sqlserver_performance.flow_control_persec): Rate per second of flow control.
// GAUGE(sqlserver_performance.flow_control_time_ms_persec): Rate per second of ms of flow control time.
// GAUGE(sqlserver_performance.forwarded_records_persec): Rate per second of record forwarding.
// GAUGE(sqlserver_performance.free_list_stalls_persec): Rate per second of stalled free list.
// GAUGE(sqlserver_performance.free_space_in_tempdb_kb): Free space in KB of tempdb.
// GAUGE(sqlserver_performance.full_scans_persec): Rate per second of full scans.
// GAUGE(sqlserver_performance.index_searches_persec): Rate per second of index searches.
// GAUGE(sqlserver_performance.latch_waits_persec): Rate per second of latch waits.
// GAUGE(sqlserver_performance.lazy_writes_persec): Rate per second of lazy writes.
// GAUGE(sqlserver_performance.lock_timeouts_persec): Rate per second of lock timeouts.
// GAUGE(sqlserver_performance.lock_wait_count): Number of lock waits.
// GAUGE(sqlserver_performance.lock_wait_time): Lock wait time.
// GAUGE(sqlserver_performance.lock_waits_persec): Rate per second of lock waits.
// GAUGE(sqlserver_performance.log_apply_pending_queue): Size of the log apply pending queue.
// GAUGE(sqlserver_performance.log_apply_ready_queue): Size of log apply ready queue.
// GAUGE(sqlserver_performance.log_bytes_flushed_persec): Rate per second of log bytes flushed.
// GAUGE(sqlserver_performance.log_bytes_received_persec): Rate per second of log bytes received.
// GAUGE(sqlserver_performance.log_files_size_kb): Size in KB of log file.
// GAUGE(sqlserver_performance.log_files_used_size_kb): Size in KB of log file used.
// GAUGE(sqlserver_performance.log_flush_wait_time): Time spent flushing the log.
// GAUGE(sqlserver_performance.log_flushes_persec): Rate per second of log flushes.
// GAUGE(sqlserver_performance.log_send_queue): Size of the log send queue.
// GAUGE(sqlserver_performance.logins_persec): Rate of logins per second.
// GAUGE(sqlserver_performance.logouts_persec): Rate of logouts per second.
// GAUGE(sqlserver_performance.memory_broker_clerk_size): Size of memory broker clerk.
// GAUGE(sqlserver_performance.memory_grants_outstanding): Number of outstanding memory grants.
// GAUGE(sqlserver_performance.memory_grants_pending): Number of pending memory grants.
// GAUGE(sqlserver_performance.number_of_deadlocks_persec): Rate of deadlocks per second.
// GAUGE(sqlserver_performance.page_life_expectancy): Page life expectancy.
// GAUGE(sqlserver_performance.page_lookups_persec): Rate of page look ups per second.
// GAUGE(sqlserver_performance.page_reads_persec): Rate of page reads per second.
// GAUGE(sqlserver_performance.page_splits_persec): Rate of page splits per second.
// GAUGE(sqlserver_performance.page_writes_persec): Rate of page writes per second.
// GAUGE(sqlserver_performance.percent_log_used): Percentage of log used.
// GAUGE(sqlserver_performance.processes_blocked): Number of blocked processes.
// GAUGE(sqlserver_performance.queued_request_count): Number of queued requests.
// GAUGE(sqlserver_performance.queued_requests): Average number of queued requests.
// GAUGE(sqlserver_performance.readahead_pages_persec): Rate per second of read ahead pages.
// GAUGE(sqlserver_performance.receives_from_replica_persec): Rate receives from replicas per second.
// GAUGE(sqlserver_performance.recovery_queue): Size of recovery queue.
// GAUGE(sqlserver_performance.redone_bytes_persec): Rate of redone bytes per second.
// GAUGE(sqlserver_performance.reduced_memory_grant_count): Number of reduced memory grants.
// GAUGE(sqlserver_performance.request_count): Number of requests.
// GAUGE(sqlserver_performance.requests_completed_persec): Rate of completed requests per second.
// GAUGE(sqlserver_performance.resent_messages_persec): Rate of resent messages per second.
// GAUGE(sqlserver_performance.sends_to_replica_persec): Rate of sends to replicas per second.
// GAUGE(sqlserver_performance.sends_to_transport_persec): Rate of sends to transports per second.
// GAUGE(sqlserver_performance.sql_compilations_persec): Rate of sql compilations per second.
// GAUGE(sqlserver_performance.sql_re-compilations_persec): Rate of sql recompilations per sec.
// GAUGE(sqlserver_performance.target_server_memory_kb): Size of target server memory in KB.
// GAUGE(sqlserver_performance.temp_tables_creation_rate): Rate of temporary table creations.
// GAUGE(sqlserver_performance.temp_tables_for_destruction): Number of temporary tables marked for destruction.
// GAUGE(sqlserver_performance.total_server_memory_kb): Total server memory in KB.
// GAUGE(sqlserver_performance.transaction_delay): Number of delayed transactions.
// GAUGE(sqlserver_performance.transactions_persec): Rate of transactions per second.
// GAUGE(sqlserver_performance.used_memory_kb): Used memory in KB.
// GAUGE(sqlserver_performance.user_connections): Number of user connections.
// GAUGE(sqlserver_performance.version_store_size_kb): Size of the version store in KB.
// GAUGE(sqlserver_performance.write_transactions_persec): Rate of write transactions per second.
// GAUGE(sqlserver_performance.xtp_memory_used_kb): Size of xtp memory used in KB.
// GAUGE(sqlserver_server_properties.available_storage_mb): Available storage in MB.
// GAUGE(sqlserver_server_properties.cpu_count): Number of cpus.
// GAUGE(sqlserver_server_properties.db_offline): Number of offline databases.
// GAUGE(sqlserver_server_properties.db_online): Number of online databases.
// GAUGE(sqlserver_server_properties.db_recovering): Number of databases recovering.
// GAUGE(sqlserver_server_properties.db_recoveryPending): Number of databases pending recovery.
// GAUGE(sqlserver_server_properties.db_restoring): Number of databases restoring.
// GAUGE(sqlserver_server_properties.db_suspect): Number of suspect databases.
// GAUGE(sqlserver_server_properties.engine_edition): Sql server engine edition version.
// GAUGE(sqlserver_server_properties.server_memory): Amount of memory on the sql server.
// GAUGE(sqlserver_server_properties.total_storage_mb): Amount of storage in MB of the sql server.
// GAUGE(sqlserver_server_properties.uptime): Uptime of the sql server.
// GAUGE(sqlserver_waitstats.max_wait_time_ms): Maximum time in millisecond spent waiting.
// GAUGE(sqlserver_waitstats.resource_wait_ms): Time in milliseconds spent waiting on a resource.
// GAUGE(sqlserver_waitstats.signal_wait_time_ms): Time in milliseconds waiting on a signal.
// GAUGE(sqlserver_waitstats.wait_time_ms): Time in milliseconds waiting.
// GAUGE(sqlserver_waitstats.waiting_tasks_count): Time in milliseconds
