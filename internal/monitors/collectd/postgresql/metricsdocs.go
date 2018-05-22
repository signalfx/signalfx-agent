package postgresql

// GAUGE(pg_blks.heap_hit): Number of buffer hits

// GAUGE(pg_blks.heap_read): Number of disk blocks read

// GAUGE(pg_blks.idx_hit): Number of index buffer hits

// GAUGE(pg_blks.idx_read): Number of index blocks read

// GAUGE(pg_blks.tidx_hit): Number of TOAST index buffer hits

// GAUGE(pg_blks.tidx_read): Number of TOAST index blocks read

// GAUGE(pg_blks.toast_hit): Number of TOAST buffer hits

// GAUGE(pg_blks.toast_read): Number of disk blocks read

// GAUGE(pg_db_size): Size of the database on disk, in bytes

// GAUGE(pg_n_tup_c.del): Number of delete operations

// GAUGE(pg_n_tup_c.hot_upd): Number of update operations not requiring index update

// GAUGE(pg_n_tup_c.ins): Number of insert operations

// GAUGE(pg_n_tup_c.upd): Number of update operations

// GAUGE(pg_n_tup_g.dead): Number of dead rows in the database

// GAUGE(pg_n_tup_g.live): Number of live rows in the database

// GAUGE(pg_numbackends): Number of server processes

// GAUGE(pg_scan.idx): Number of index scans

// GAUGE(pg_scan.idx_tup_fetch): Number of rows read from index scans

// GAUGE(pg_scan.seq): Number of sequential scans

// GAUGE(pg_scan.seq_tup_read): Number of rows read from sequential scans

// GAUGE(pg_xact.commit): Number of commits

// GAUGE(pg_xact.num_deadlocks): Number of deadlocks detected by the database

// GAUGE(pg_xact.rollback): Number of rollbacks
