package postgresql

import "github.com/signalfx/signalfx-agent/pkg/monitors/sql"

// Queries that get metrics about the entire server instance and do not need to
// be run on a per-database basis.
var defaultServerQueries = []sql.Query{
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
		Query: `SELECT datname as database, usename as user, SUM(calls) as total_calls, SUM(total_time) as total_time FROM pg_stat_statements INNER JOIN pg_stat_database ON pg_stat_statements.dbid = pg_stat_database.datid INNER JOIN pg_user ON pg_stat_statements.userid = pg_user.usesysid GROUP BY pg_stat_database.datname, pg_user.usename;`,
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

var makeDefaultStatementsQueries = func(limit int) []sql.Query {
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
			Query:  `SELECT datname as database, usename as user, queryid, query, total_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY total_time DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= $1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
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
			Query:  `SELECT datname as database, usename as user, queryid, query, (total_time / calls) AS average_time FROM (SELECT * FROM (SELECT ROW_NUMBER() OVER (PARTITION BY dbid ORDER BY total_time / calls DESC) AS r, s.* FROM pg_stat_statements s) q WHERE q.r <= $1) p, pg_stat_database d, pg_user u WHERE p.dbid = d.datid AND p.userid = u.usesysid;`,
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
