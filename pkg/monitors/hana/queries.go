package hana

import "github.com/signalfx/signalfx-agent/pkg/monitors/sql"

// Queries that get metrics about the entire server instance and do not need to
// be run on a per-database basis.
var defaultServerQueries = []sql.Query{
	{
		Query: `SELECT host AS hana_host, usage_type, used_size FROM m_disk_usage WHERE used_size >= 0;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.disk.used_size",
				ValueColumn:      "used_size",
				DimensionColumns: []string{"hana_host", "usage_type"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, SUM(total_device_size) AS total_size FROM (SELECT device_id, host, MAX(total_device_size) AS total_device_size FROM m_disks GROUP BY device_id, host) GROUP BY host;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.disk.total_size",
				ValueColumn:      "total_size",
				DimensionColumns: []string{"hana_host"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, service_name, process_cpu, open_file_count FROM m_service_statistics;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.service.cpu.utilization",
				ValueColumn:      "process_cpu",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.file.open",
				ValueColumn:      "open_file_count",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, free_physical_memory, used_physical_memory, free_swap_space, used_swap_space, allocation_limit, instance_total_memory_used_size, instance_total_memory_allocated_size, instance_code_size, instance_shared_memory_allocated_size, open_file_count, total_cpu_user_time, total_cpu_system_time, total_cpu_wio_time, total_cpu_idle_time FROM m_host_resource_utilization;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.host.memory.physical.free",
				ValueColumn:      "free_physical_memory",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.physical.used",
				ValueColumn:      "used_physical_memory",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.swap.free",
				ValueColumn:      "free_swap_space",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.swap.used",
				ValueColumn:      "used_swap_space",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.allocation_limit",
				ValueColumn:      "allocation_limit",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.total_used",
				ValueColumn:      "instance_total_memory_used_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.total_allocated",
				ValueColumn:      "instance_total_memory_allocated_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.code",
				ValueColumn:      "instance_code_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.memory.shared",
				ValueColumn:      "instance_shared_memory_allocated_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.file.open",
				ValueColumn:      "open_file_count",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.host.cpu.user",
				ValueColumn:      "total_cpu_user_time",
				DimensionColumns: []string{"hana_host"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.host.cpu.system",
				ValueColumn:      "total_cpu_system_time",
				DimensionColumns: []string{"hana_host"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.host.cpu.wio",
				ValueColumn:      "total_cpu_wio_time",
				DimensionColumns: []string{"hana_host"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.host.cpu.idle",
				ValueColumn:      "total_cpu_idle_time",
				DimensionColumns: []string{"hana_host"},
				IsCumulative:     true,
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, service_name, logical_memory_size, physical_memory_size, code_size, stack_size, heap_memory_allocated_size, heap_memory_used_size, shared_memory_allocated_size, shared_memory_used_size, allocation_limit, effective_allocation_limit, total_memory_used_size FROM m_service_memory;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.service.memory.logical",
				ValueColumn:      "logical_memory_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.physical",
				ValueColumn:      "physical_memory_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.code",
				ValueColumn:      "code_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.stack",
				ValueColumn:      "stack_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.heap.allocated",
				ValueColumn:      "heap_memory_allocated_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.heap.used",
				ValueColumn:      "heap_memory_used_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.shared.allocated",
				ValueColumn:      "shared_memory_allocated_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.shared.used",
				ValueColumn:      "shared_memory_used_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.allocation_limit",
				ValueColumn:      "allocation_limit",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.allocation_limit_effective",
				ValueColumn:      "effective_allocation_limit",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
			{
				MetricName:       "sap.hana.service.memory.total_used",
				ValueColumn:      "total_memory_used_size",
				DimensionColumns: []string{"hana_host", "service_name"},
			},
		},
	},
	{
		Query: `SELECT services.host AS hana_host, services.service_name AS service_name, memory.component AS component_name, memory.used_memory_size AS used_memory_size FROM m_service_component_memory AS memory JOIN m_services AS services ON memory.host = services.host AND memory.port = services.port;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.service.component.memory.used",
				ValueColumn:      "used_memory_size",
				DimensionColumns: []string{"hana_host", "service_name", "component_name"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, COUNT(*) AS statement_count, SUM(recompile_count) AS recompile_count, SUM(execution_count) AS execution_count, TO_DOUBLE(AVG(avg_execution_time)) AS avg_execution_time, MAX(max_execution_time) AS max_execution_time, SUM(total_execution_time) AS total_execution_time, TO_DOUBLE(AVG(avg_execution_memory_size)) AS avg_execution_memory_size, MAX(max_execution_memory_size) AS max_execution_memory_size, SUM(total_execution_memory_size) AS total_execution_memory_size FROM m_active_statements GROUP BY host;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.statement.active.count",
				ValueColumn:      "statement_count",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.recompile.count",
				ValueColumn:      "recompile_count",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.count",
				ValueColumn:      "execution_count",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.time.mean",
				ValueColumn:      "avg_execution_time",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.time.sum",
				ValueColumn:      "total_execution_time",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.time.max",
				ValueColumn:      "max_execution_time",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.memory.mean",
				ValueColumn:      "avg_execution_memory_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.memory.max",
				ValueColumn:      "max_execution_memory_size",
				DimensionColumns: []string{"hana_host"},
			},
			{
				MetricName:       "sap.hana.statement.active.execution.memory.sum",
				ValueColumn:      "total_execution_memory_size",
				DimensionColumns: []string{"hana_host"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host,statement_hash,db_user,schema_name,app_user,operation,SUM(duration_microsec) AS total_duration_microsec,SUM(records) AS total_records,SUM(cpu_time) AS total_cpu_time,SUM(lock_wait_duration) AS total_lock_wait_duration,COUNT(*) AS count FROM m_expensive_statements WHERE start_time > ADD_DAYS(CURRENT_TIMESTAMP , -1) AND (host , schema_name , statement_hash) IN ( SELECT host , schema_name , statement_hash FROM (SELECT * , rank() OVER (PARTITION BY host , schema_name ORDER BY duration_microsec DESC) AS rank FROM (SELECT host , schema_name , statement_hash , MAX (duration_microsec) AS duration_microsec FROM m_expensive_statements WHERE start_time > ADD_DAYS(CURRENT_TIMESTAMP , -1) GROUP BY host , schema_name , statement_hash)) WHERE rank < 10 )GROUP BY host, statement_hash, db_user, schema_name, app_user, operation;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.statement.expensive.count",
				ValueColumn:      "count",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.statement.expensive.duration",
				ValueColumn:      "total_duration_microsec",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
			},
			{
				MetricName:       "sap.hana.statement.expensive.records",
				ValueColumn:      "total_records",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.statement.expensive.cpu_time",
				ValueColumn:      "total_cpu_time",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
				IsCumulative:     true,
			},
			{
				MetricName:       "sap.hana.statement.expensive.lock_wait_duration",
				ValueColumn:      "total_lock_wait_duration",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
				IsCumulative:     true,
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, statement_hash, db_user, schema_name, app_user, operation, COUNT(*) AS errors FROM m_expensive_statements WHERE error_code <> 0 GROUP BY host, statement_hash, db_user, schema_name, app_user, operation;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.statement.expensive.errors",
				ValueColumn:      "errors",
				DimensionColumns: []string{"hana_host", "statement_hash", "db_user", "schema_name", "app_user", "operation"},
				IsCumulative:     true,
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, connection_status, COUNT(*) AS count, SUM(memory_size_per_connection) AS memory_size, SUM(fetched_record_count) AS fetched_record_count, SUM(affected_record_count) AS affected_record_count, SUM(sent_message_size) AS sent_message_size, SUM(sent_message_count) AS sent_message_count, SUM(received_message_size) AS received_message_size, SUM(received_message_count) AS received_message_count FROM m_connections GROUP BY host, connection_status HAVING connection_status != '';`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.connection.count",
				ValueColumn:      "count",
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.memory.allocated",
				ValueColumn:      "memory_size",
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.record.fetched",
				ValueColumn:      "fetched_record_count",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.record.affected",
				ValueColumn:      "affected_record_count",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.message.sent.size",
				ValueColumn:      "sent_message_size",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.message.sent.count",
				ValueColumn:      "sent_message_count",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.message.received.size",
				ValueColumn:      "received_message_size",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
			{
				MetricName:       "sap.hana.connection.message.received.count",
				ValueColumn:      "received_message_count",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "connection_status"},
			},
		},
	},
	{
		Query: `SELECT host AS hana_host, type, SUM(total_reads) AS total_reads, SUM(total_trigger_async_reads) AS total_trigger_async_reads, SUM(total_failed_reads) AS total_failed_reads, SUM(total_read_size) AS total_read_size, SUM(total_read_time) AS total_read_time, SUM(total_appends) AS total_appends, SUM(total_writes) AS total_writes, SUM(total_trigger_async_writes) AS total_trigger_async_writes, SUM(total_failed_writes) AS total_failed_writes, SUM(total_write_size) AS total_write_size, SUM(total_write_time) AS total_write_time, SUM(total_io_time) AS total_io_time FROM m_volume_io_total_statistics GROUP BY host, type;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.io.read.count",
				ValueColumn:      "total_reads",
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.read.async.count",
				ValueColumn:      "total_trigger_async_reads",
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.read.failed",
				ValueColumn:      "total_failed_reads",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.read.size",
				ValueColumn:      "total_read_size",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.read.time",
				ValueColumn:      "total_read_time",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.append.count",
				ValueColumn:      "total_appends",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.write.count",
				ValueColumn:      "total_writes",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.write.async.count",
				ValueColumn:      "total_trigger_async_writes",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.write.failed",
				ValueColumn:      "total_failed_writes",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.write.size",
				ValueColumn:      "total_write_size",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.write.time",
				ValueColumn:      "total_write_time",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
			{
				MetricName:       "sap.hana.io.total.time",
				ValueColumn:      "total_io_time",
				IsCumulative:     true,
				DimensionColumns: []string{"hana_host", "type"},
			},
		},
	},
	{
		Query: `SELECT schema_name, table_name, table_type,	record_count, table_size FROM m_tables WHERE schema_name NOT IN ('SYS', 'SAP_PA_APL', 'BROKER_PO_USER') AND schema_name NOT LIKE '_SYS_%';`,
		Metrics: []sql.Metric{
			{
				MetricName:       "sap.hana.table.record.count",
				ValueColumn:      "record_count",
				DimensionColumns: []string{"schema_name", "table_name", "table_type"},
			},
			{
				MetricName:       "sap.hana.table.size",
				ValueColumn:      "table_size",
				DimensionColumns: []string{"schema_name", "table_name", "table_type"},
			},
		},
	},
}
