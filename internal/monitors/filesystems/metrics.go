package filesystems

// GAUGE(df_complex.free): Free disk space in bytes

// GAUGE(df_complex.used): Used disk space in bytes

// GAUGE(df_inodes.free): (Linux Only) Number of inodes that are free.  This is
// is only reported if the configuration option `reportInodes` is set to `true`.

// GAUGE(df_inodes.used): (Linux Only) Number of inodes that are used.  This is
// only reported if the configuration option `reportInodes` is set to `true`.

// GAUGE(percent_bytes.free): Free disk space on the file system,
// expressed as a percentage.

// GAUGE(percent_bytes.used): Used disk space on the file system,
// expressed as a percentage.

// GAUGE(percent_inodes.free): (Linux Only) Free inodes on the file system, expressed
// as a percentage.  This is only reported if the configuration option `reportInodes`
// is set to `true`.

// GAUGE(percent_inodes.used): (Linux Only) Used inodes on the file system, expressed
// as a percentage.  This is only reported if the configuration option `reportInodes`
// is set to `true`.

// GAUGE(disk.summary_utilization): Percent of disk space utilized on all
// volumes on this host. This metric reports with plugin dimension set to
// "signalfx-metadata".

// GAUGE(disk.utilization): Percent of disk used on this volume. This metric
// reports with plugin dimension set to "signalfx-metadata".
