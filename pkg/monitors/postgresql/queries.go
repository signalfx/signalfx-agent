package postgresql

import (
	"fmt"

	"github.com/signalfx/signalfx-agent/pkg/monitors/sql"
)

// Queries that get metrics about the entire server instance and do not need to
// be run on a per-database basis.
func defaultServerQueries(totalTimeColumn string) []sql.Query {
	return []sql.Query{
		{
			Query: `select oid, datname as database from pg_database;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_database",
					ValueColumn:      "database",
					DimensionColumns: []string{"oid"},
					isCumulative: false,
				},
			},
		},
		{
			Query: `SELECT I.table_catalog as database, C.relname AS 'table', pg_size_pretty(pg_table_size(C.oid)) AS 'table_size' FROM information_schema.tables I, pg_class C LEFT JOIN pg_namespace N ON (N.oid = C.relnamespace) WHERE nspname NOT IN ('pg_catalog', 'information_schema') AND nspname !~ '^pg_toast' AND relkind IN ('r') AND I.table_name = C.relname ORDER BY pg_table_size(C.oid) DESC;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_table_size",
					ValueColumn:      "table_size",
					DimensionColumns: []string{"database, table"},
					isCumulative: false,
				},
			},
		},
		{
			Query: `SELECT datname as database, usename as user, SUM(calls) as total_calls, total_exec_time + total_plan_time as total_time FROM pg_stat_statements INNER JOIN pg_stat_database ON pg_stat_statements.dbid = pg_stat_database.datid INNER JOIN pg_user ON pg_stat_statements.userid = pg_user.usesysid GROUP BY pg_stat_database.datname, pg_user.usename, pg_stat_statements.total_exec_time, pg_stat_statements.total_plan_time;`
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_user_query_count",
					ValueColumn:      "total_calls",
					DimensionColumns: []string{"database", "user"},
					IsCumulative:     false,
				},
				{
					MetricName:       "postgres_user_query_time",
					ValueColumn:      "total_time",
					DimensionColumns: []string{"database", "user"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `select datid as database_id, state, datname as database, pid, sessions, query_id, query, state_change - query_start as time_taken FROM pg_stat_activity WHERE state IS NOT NULL  and query IS NOT NULL GROUP BY pg_stat_activity.state, pg_stat_activity.datname, datid,pid,query_id,query, state_change, query_start;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_sessions",
					ValueColumn: "sessions",
					DimensionColumns: []string{"database_id", "database", "query_id", "query"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_longest_running_queries",
					ValueColumn: "time_taken",
					DimensionColumns: []string{"database_id", "database","sessions", "query_id", "query"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `select d.datname AS database, pg_size_pretty(pg_database_size(d.datname)) as database_size from pg_database d;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_database_size",
					ValueColumn: "database_size",
				},
			},
		},
		{
			Query: `SELECT relname as table, pg_stat_all_tables.schemaname, n_tup_ins, n_tup_upd, n_tup_del, n_live_tup, n_dead_tup, n_tup_hot_upd, seq_scan, COALESCE(idx_scan, 0) as idx_scan, pg_relation_size(relid) as size, 'user' as type from pg_stat_all_tables WHERE idx_scan IS NOT NULL and pg_stat_all_tables.schemaname = 'public';`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_rows_inserted",
					ValueColumn:      "n_tup_ins",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_rows_updated",
					ValueColumn:      "n_tup_upd",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_rows_hot_updated",
					ValueColumn:      "n_tup_hot_upd",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_rows_deleted",
					ValueColumn:      "n_tup_del",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_sequential_scans",
					ValueColumn:      "seq_scan",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_index_scans",
					ValueColumn:      "idx_scan",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_table_size",
					ValueColumn:      "size",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_live_rows",
					ValueColumn:      "n_live_tup",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_dead_rows",
					ValueColumn:      "n_dead_tup",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `select checkpoints_timed as scheduled_checkpoint, checkpoints_req as requested_checkpoint, buffers_checkpoint, buffers_backend, buffers_clean from pg_stat_bgwriter;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_scheduled_checkpoints",
					ValueColumn: "scheduled_checkpoint",
					DimensionColumns: nil,
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_requested_checkpoints",
					ValueColumn: "requested_checkpoint",
					DimensionColumns: nil,
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_buffers_checkpoints",
					ValueColumn: "buffers_checkpoint",
					DimensionColumns: nil,
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_buffers_backend",
					ValueColumn: "buffers_backend",
					DimensionColumns: nil,
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_buffers_clean",
					ValueColumn: "buffers_clean",
					DimensionColumns: nil,
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `select count(pid) as locks, database as database_id, relation, a.relname as database, mode, locktype from pg_locks as l, pg_class as a where l.relation = a.oid  and a.relname ='search_hit' group by database, relation, a.relname, mode, locktype;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_database_locks",
					ValueColumn: "locks",
					DimensionColumns: []string{"locktype", "relation", "database_id","database", "mode"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `select pid, state, usename as user, query, query_start, datname as database, datid as database_id from pg_stat_activity where pid in (select pid from pg_locks l join pg_class t on l.relation = t.oid and t.relkind = 'r' where t.relname = 'search_hit');`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_database_locks_queries",
					ValueColumn: "pid",
					DimensionColumns: []string{"database_id", "database", "query", "state", "user"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `select datid as database_id, datname as database, sessions, numbackends as backends, xact_commit as commits, xact_rollback as rollbacks, conflicts as conflicts, deadlocks, temp_files as temporary_files, temp_bytes as temporary_bytes, xact_commit + xact_rollback as total_transactions from pg_stat_database as d where d.datname <> '';`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_backends",
					ValueColumn: "backends",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_total_transactions",
					ValueColumn: "total_transactions",
					DimensionColumns: []string{"database_id", "database","commits","rollbacks"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_total_sessions",
					ValueColumn: "sessions",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_rollbacks",
					ValueColumn: "rollbacks",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_conflicts",
					ValueColumn: "conflicts",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_deadlocks",
					ValueColumn: "deadlocks",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_temporary_files",
					ValueColumn: "temporary_files",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_temporary_bytes",
					ValueColumn: "temporary_bytes",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `Select datid as database_id, datname as database, tup_returned as returned, tup_fetched as fetched, tup_inserted as inserted, tup_updated as updated, tup_deleted as deleted from pg_stat_database;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_database_returned",
					ValueColumn: "returned",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_database_fetched",
					ValueColumn: "fetched",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_database_inserted",
					ValueColumn: "inserted",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_database_updated",
					ValueColumn: "updated",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_database_deleted",
					ValueColumn: "deleted",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `SELECT datid as database_id, datname as database, 100 * blks_hit / (blks_hit + blks_read) as cache_hit_ratio, 100 * xact_commit / (xact_commit + xact_rollback) as commit_ratio FROM pg_stat_database WHERE (xact_commit + xact_rollback) > 0 and (blks_hit + blks_read) > 0 and datname <> '';`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_commit_ratio",
					ValueColumn: "commit_ratio",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
				{
					MetricName:  "postgres_cache_hit_ratio",
					ValueColumn: "cache_hit_ratio",
					DimensionColumns: []string{"database_id", "database"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `SELECT schemaname, relname as table, indexrelname as index, (idx_blks_hit*1.0 / GREATEST(idx_blks_hit + idx_blks_read, 1)) as hit_ratio, 'user' as type FROM pg_statio_user_indexes;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_table_block_hit_ratio",
					ValueColumn: "hit_ratio",
					DimensionColumns: []string{"index", "table", "schemaname", "type"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `SELECT datname as database, usename as user, queryid, query, calls FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY calls DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= 1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_queries_calls",
					ValueColumn: "calls",
					DimensionColumns: []string{"database", "queryid", "user", "query"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `SELECT datname as database, usename as user, queryid, query, ((total_exec_time + total_plan_time) / calls) AS average_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY ((total_exec_time + total_plan_time) / calls) DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= 1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_queries_average_time",
					ValueColumn: "average_time",
					DimensionColumns: []string{"database", "queryid", "query"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `SELECT datname as database, usename as user, queryid, query, (total_exec_time + total_plan_time) as total_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY (total_exec_time + total_plan_time) DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= 1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_queries_total_time",
					ValueColumn: "total_time",
					DimensionColumns: []string{"database", "queryid", "user", "query"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `select datname as database, sessions, sessions_abandoned, sessions_fatal from pg_stat_database;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_sessions",
					ValueColumn: "sessions",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
				{
					MetricName:  "postgres_sessions_fatal",
					ValueColumn: "sessions_fatal",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
				{
					MetricName:  "postgres_sessions_abondoned",
					ValueColumn: "sessions_abondoned",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `SELECT COUNT(*) as count, state, datname as database FROM pg_stat_activity WHERE state IS NOT NULL GROUP BY pg_stat_activity.state, pg_stat_activity.datname;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_sessions",
					ValueColumn:      "count",
					DimensionColumns: []string{"state", "database"},
				},
			},
		},
		{
			Query: `SELECT datname as database, (blks_hit*1.0 / GREATEST(blks_read + blks_hit, 1)) as blks_hit_ratio, deadlocks FROM pg_stat_database WHERE blks_read > 0;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_block_hit_ratio",
					ValueColumn:      "blks_hit_ratio",
					DimensionColumns: []string{"database"},
				},
				{
					MetricName:       "postgres_deadlocks",
					ValueColumn:      "deadlocks",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: fmt.Sprintf(`SELECT datname as database, usename as user, SUM(calls) as total_calls, SUM(%s) as total_time FROM pg_stat_statements INNER JOIN pg_stat_database ON pg_stat_statements.dbid = pg_stat_database.datid INNER JOIN pg_user ON pg_stat_statements.userid = pg_user.usesysid GROUP BY pg_stat_database.datname, pg_user.usename;`, totalTimeColumn), //nolint,gosec // column name will only be total_time or total_exec_time.
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_query_count",
					ValueColumn:      "total_calls",
					DimensionColumns: []string{"database", "user"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_query_time",
					ValueColumn:      "total_time",
					DimensionColumns: []string{"database", "user"},
					IsCumulative:     true,
				},
			},
		},
		{
			Query: `WITH max_con AS (SELECT setting::float FROM pg_settings WHERE name = 'max_connections') SELECT COUNT(*)/MAX(setting) AS pct_connections FROM pg_stat_activity, max_con;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_pct_connections",
					ValueColumn: "pct_connections",
				},
			},
		},
		{
			Query: `SELECT COUNT(*) AS locks FROM pg_locks WHERE NOT granted;`,
			Metrics: []sql.Metric{
				{
					MetricName:  "postgres_locks",
					ValueColumn: "locks",
				},
			},
		},
		{
			Query: `SELECT datname AS database, xact_commit, xact_rollback, conflicts FROM pg_stat_database;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_conflicts",
					ValueColumn:      "conflicts",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_xact_commits",
					ValueColumn:      "xact_commit",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_xact_rollbacks",
					ValueColumn:      "xact_rollback",
					DimensionColumns: []string{"database"},
					IsCumulative:     true,
				},
			},
		},
	}
}

var makeDefaultDBQueries = func(dbname string) []sql.Query {
	return []sql.Query{
		{
			Query: `SELECT tablespace, relname as table, pg_stat_user_tables.schemaname, n_live_tup, n_tup_ins, n_tup_upd, n_tup_del, seq_scan, COALESCE(idx_scan, 0) as idx_scan, pg_relation_size(relid) as size, 'user' as type from pg_stat_user_tables INNER JOIN pg_tables ON (pg_stat_user_tables.relname = pg_tables.tablename AND pg_stat_user_tables.schemaname = pg_tables.schemaname) WHERE idx_scan IS NOT NULL AND pg_relation_size(relid) IS NOT NULL;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_rows_inserted",
					ValueColumn:      "n_tup_ins",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_rows_updated",
					ValueColumn:      "n_tup_upd",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_rows_deleted",
					ValueColumn:      "n_tup_del",
					DimensionColumns: []string{"schemaname", "table", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_sequential_scans",
					ValueColumn:      "seq_scan",
					DimensionColumns: []string{"table", "schemaname", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_index_scans",
					ValueColumn:      "idx_scan",
					DimensionColumns: []string{"table", "schemaname", "type", "tablespace"},
					IsCumulative:     true,
				},
				{
					MetricName:       "postgres_table_size",
					ValueColumn:      "size",
					DimensionColumns: []string{"table", "schemaname", "type", "tablespace"},
					IsCumulative:     false,
				},
				{
					MetricName:       "postgres_live_rows",
					ValueColumn:      "n_live_tup",
					DimensionColumns: []string{"table", "schemaname", "type", "tablespace"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query:  `SELECT pg_database_size($1) as size;`,
			Params: []interface{}{dbname},
			Metrics: []sql.Metric{
				{
					MetricName:   "postgres_database_size",
					ValueColumn:  "size",
					IsCumulative: false,
				},
			},
		},
		{
			Query: `SELECT schemaname, relname as table, (heap_blks_hit*1.0 / GREATEST(heap_blks_read+heap_blks_hit, 1)) as block_hit_ratio, 'user' as type from pg_statio_user_tables;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_block_hit_ratio",
					ValueColumn:      "block_hit_ratio",
					DimensionColumns: []string{"table", "schemaname", "type"},
					IsCumulative:     false,
				},
			},
		},
		{
			Query: `SELECT schemaname, relname as table, indexrelname as index, (idx_blks_hit*1.0 / GREATEST(idx_blks_hit + idx_blks_read, 1)) as hit_ratio, 'user' as type FROM pg_statio_user_indexes;`,
			Metrics: []sql.Metric{
				{
					MetricName:       "postgres_block_hit_ratio",
					ValueColumn:      "hit_ratio",
					DimensionColumns: []string{"index", "table", "schemaname", "type"},
					IsCumulative:     false,
				},
			},
		},
	}

}

var makeDefaultStatementsQueries = func(limit int, totalTimeColumn string) []sql.Query {
	return []sql.Query{
		{
			Query:  `SELECT datname as database, usename as user, queryid, query, calls FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY calls DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= $1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
			Params: []interface{}{limit},
			Metrics: []sql.Metric{
				{
					MetricName:               "postgres_queries_calls",
					ValueColumn:              "calls",
					DimensionColumns:         []string{"database", "user", "queryid"},
					IsCumulative:             true,
					DimensionPropertyColumns: map[string][]string{"queryid": {"query"}},
				},
			},
		},
		{
			Query:  fmt.Sprintf(`SELECT datname as database, usename as user, queryid, query, %s as total_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY %s DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= $1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`, totalTimeColumn, totalTimeColumn), //nolint,gosec // column name will only be total_time or total_exec_time.
			Params: []interface{}{limit},
			Metrics: []sql.Metric{
				{
					MetricName:               "postgres_queries_total_time",
					ValueColumn:              "total_time",
					DimensionColumns:         []string{"database", "user", "queryid"},
					IsCumulative:             true,
					DimensionPropertyColumns: map[string][]string{"queryid": {"query"}},
				},
			},
		},
		{
			Query:  fmt.Sprintf(`SELECT datname as database, usename as user, queryid, query, (%s / calls) AS average_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY %s / calls DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= $1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`, totalTimeColumn, totalTimeColumn), //nolint,gosec // column name will only be total_time or total_exec_time.
			Params: []interface{}{limit},
			Metrics: []sql.Metric{
				{
					MetricName:               "postgres_queries_average_time",
					ValueColumn:              "average_time",
					DimensionColumns:         []string{"database", "user", "queryid"},
					IsCumulative:             true,
					DimensionPropertyColumns: map[string][]string{"queryid": {"query"}},
				},
			},
		},
	}
}

var defaultReplicationQueries = []sql.Query{
	{
		Query: `SELECT GREATEST (0, (EXTRACT (EPOCH FROM now() - pg_last_xact_replay_timestamp()))) AS lag, CASE WHEN pg_is_in_recovery() THEN 'standby' ELSE 'master' END AS replication_role;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "postgres_replication_lag",
				ValueColumn:      "lag",
				DimensionColumns: []string{"replication_role"},
			},
		},
	},
	{
		Query: `SELECT slot_name, slot_type, database, case when active then 1 else 0 end AS active FROM pg_replication_slots;`,
		Metrics: []sql.Metric{
			{
				MetricName:       "postgres_replication_state",
				ValueColumn:      "active",
				DimensionColumns: []string{"slot_name", "slot_type", "database"},
			},
		},
	},
}
