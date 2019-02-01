package procstat

// GAUGE(procstat.cpu_usage): CPU used by the process.
// GAUGE(procstat.memory_data): VMData memory used by the process.
// GAUGE(procstat.memory_locked): VMLocked memory used by the process.
// GAUGE(procstat.memory_rss): VMRSS memory used by the process.
// GAUGE(procstat.memory_stack): VMStack memory used by the process.
// GAUGE(procstat.memory_swap): VMSwap memory used by the process.
// GAUGE(procstat.memory_vms): VMS memory used by the process.
// GAUGE(procstat.num_threads): Number of threads used by the process.
// GAUGE(procstat.read_bytes): Number of bytes read by the process.  This may require the agent to be running as root.
// GAUGE(procstat.read_count): Number of read operations by the process.  This may require the agent to be running as root.
// GAUGE(procstat.write_bytes): Number of bytes written by the process.  This may require the agent to be running as root.
// GAUGE(procstat.write_count): Number of write operations by the process.  This may require the agent to be running as root.
// GAUGE(procstat.cpu_time): Amount of cpu time consumed by the process.
// GAUGE(procstat.involuntary_context_switches): Number of involuntary context switches.
// GAUGE(procstat.nice_priority): Nice priority number of the process.
// GAUGE(procstat.num_fds): Number of file descriptors.  This may require the agent to be running as root.
// GAUGE(procstat.realtime_priority): Real time priority of the process.
// GAUGE(procstat.rlimit_cpu_time_hard): The hard cpu rlimit.
// GAUGE(procstat.rlimit_cpu_time_soft): The soft cpu rlimit.
// GAUGE(procstat.rlimit_file_locks_hard): The hard file lock rlimit.
// GAUGE(procstat.rlimit_file_locks_soft): The soft file lock rlimit.
// GAUGE(procstat.rlimit_memory_data_hard): The hard data memory rlimit.
// GAUGE(procstat.rlimit_memory_data_soft): The soft data memory rlimit.
// GAUGE(procstat.rlimit_memory_locked_hard): The hard locked memory rlimit.
// GAUGE(procstat.rlimit_memory_locked_soft): The soft locked memory rlimit.
// GAUGE(procstat.rlimit_memory_rss_hard): The hard rss memory rlimit.
// GAUGE(procstat.rlimit_memory_rss_soft): The soft rss memory rlimit.
// GAUGE(procstat.rlimit_memory_stack_hard): The hard stack memory rlimit.
// GAUGE(procstat.rlimit_memory_stack_soft): The soft stack memory rlimit.
// GAUGE(procstat.rlimit_memory_vms_hard): The hard vms memory rlimit.
// GAUGE(procstat.rlimit_memory_vms_soft): The soft vms memory rlimit.
// GAUGE(procstat.rlimit_nice_priority_hard): The hard nice priority rlimit.
// GAUGE(procstat.rlimit_nice_priority_soft): The soft nice priority rlimit.
// GAUGE(procstat.rlimit_num_fds_hard): The hard file descriptor rlimit.
// GAUGE(procstat.rlimit_num_fds_soft): The soft file descriptor rlimit.
// GAUGE(procstat.rlimit_realtime_priority_hard): The hard realtime priority rlimit.
// GAUGE(procstat.rlimit_realtime_priority_soft): The soft realtime priority rlimit.
// GAUGE(procstat.rlimit_signals_pending_hard): The hard pending signal rlimit.
// GAUGE(procstat.rlimit_signals_pending_soft): The soft pendidng signal rlimit.
// GAUGE(procstat.signals_pending): The number of signals pending.
// GAUGE(procstat_lookup.pid_count): The number of pids. This metric emits with the plugin dimension set to "procstat_lookup".
