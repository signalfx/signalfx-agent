package utilization

// GAUGE(disk.utilization): This metric shows the amount of disk space in use on
// this volume, as a percentage of total disk space available on the volume.

// GAUGE(df_complex.free): This metric measures free disk space in bytes on this
// file system.

// GAUGE(df_complex.used): This metric measures used disk space in bytes on this
// file system.

// GAUGE(memory.utilization): This metric shows the amount of memory in use on
// this machine, as a percent of total memory available.

// GAUGE(memory.free): Amount of memory that can be immediately used, in bytes.

// GAUGE(memory.used): The amount of memory consumed by processes and the
// system, in bytes.

// GAUGE(cpu.utilization): This metric shows the amount of CPU in use on this
// machine, as a percent of total CPU available.

// GAUGE(cpu.utilization_per_core): This metric shows the amount of CPU in use
// on each core, as a percent of total CPU available per core.

// GAUGE(network_interface.bytes_received.per_second): The rate of bytes
// received per second.

// GAUGE(network_interface.bytes_sent.per_second): The rate of bytes sent per
// second.

// GAUGE(network_interface.errors_received.per_second): The rate of errors
// encountered per second while receiving.

// GAUGE(network_interface.errors_sent.per_second): The rate of errors
// encountered per second while sending.

// GAUGE(disk.reads.per_second): The rate of read operations per second against
// a disk.

// GAUGE(disk.write.per_second): The rate of write operations per second against
// a disk.

// GAUGE(paging_file.pct_usage): (WINDOWS ONLY) The percentage of page file
// usage.

// GAUGE(vmpage.swapped_in.per_second): The rate of swap operations per second
// into memory.

// GAUGE(vmpage.swapped_out.per_second): The rate of swap operations per second
// out of memory.

// GAUGE(vmpage.swapped.per_second): The rate of swap operations
// (read and write) per second.
