package docker

// CUMULATIVE(blkio.io_service_bytes_recursive.async): Volume, in bytes, of asynchronous block I/O

// CUMULATIVE(blkio.io_service_bytes_recursive.read): Volume, in bytes, of reads from block devices
// CUSTOM(blkio.io_service_bytes_recursive.read): false

// CUMULATIVE(blkio.io_service_bytes_recursive.sync): Volume, in bytes, of synchronous block I/O

// CUMULATIVE(blkio.io_service_bytes_recursive.total): Total volume, in bytes, of all block I/O

// CUMULATIVE(blkio.io_service_bytes_recursive.write): Volume, in bytes, of writes to block devices
// CUSTOM(blkio.io_service_bytes_recursive.write): false

// CUMULATIVE(blkio.io_serviced_recursive.async): Number of asynchronous block I/O requests

// CUMULATIVE(blkio.io_serviced_recursive.read): Number of reads requests from block devices

// CUMULATIVE(blkio.io_serviced_recursive.sync): Number of synchronous block I/O requests

// CUMULATIVE(blkio.io_serviced_recursive.total): Total number of block I/O requests

// CUMULATIVE(blkio.io_serviced_recursive.write): Number of write requests to block devices

// GAUGE(cpu.percent): Percentage of host CPU resources used by the container

// CUMULATIVE(cpu.percpu.usage): Jiffies of CPU time spent by the container, per CPU core

// CUMULATIVE(cpu.percpu.usage): Jiffies of CPU time spent by the container, per CPU core

// CUMULATIVE(cpu.throttling_data.periods): Number of periods

// CUMULATIVE(cpu.throttling_data.throttled_periods): Number of periods throttled

// CUMULATIVE(cpu.throttling_data.throttled_time): Throttling time in nano seconds

// CUMULATIVE(cpu.usage.kernelmode): Jiffies of CPU time spent in kernel mode by the container

// CUMULATIVE(cpu.usage.system): Jiffies of CPU time used by the system
// CUSTOM(cpu.usage.system): false

// CUMULATIVE(cpu.usage.total): Jiffies of CPU time used by the container
// CUSTOM(cpu.usage.total): false

// CUMULATIVE(cpu.usage.usermode): Jiffies of CPU time spent in user mode by the container

// GAUGE(memory.percent): Percent of memory (0-100) used by the container
// relative to its limit (excludes page cache usage)

// GAUGE(memory.stats.swap): Bytes of swap memory used by container

// GAUGE(memory.stats.active_anon): Amount of memory that has been identified as active by
// the kernel. Anonymous memory is memory that is not linked to disk pages.

// GAUGE(memory.stats.active_file): Amount of active file cache memory.
// Cache memory = active_file + inactive_file + tmpfs

// GAUGE(memory.stats.cache): The amount of memory used by the processes of this control group
// that can be associated with a block on a block device. Also accounts for memory used by
// tmpfs.

// GAUGE(memory.stats.dirty): The amount of memory waiting to get written to disk

// GAUGE(memory.stats.hierarchical_memory_limit): The memory limit in place by the hierarchy cgroup

// GAUGE(memory.stats.hierarchical_memsw_limit): The memory+swap limit in place by the hierarchy cgroup

// GAUGE(memory.stats.inactive_anon): Amount of memory that has been identified as inactive by
// the kernel. Anonymous memory is memory that is not linked to disk pages.

// GAUGE(memory.stats.inactive_file): Amount of inactive file cache memory.
// Cache memory = active_file + inactive_file + tmpfs

// GAUGE(memory.stats.mapped_file): Indicates the amount of memory mapped by the processes in
// the control group. It doesn’t give you information about how much memory is used;
// it rather tells you how it is used.

// CUMULATIVE(memory.stats.pgfault): Number of times that a process of the cgroup triggered
// a page fault. Page faults occur when a process accesses part of its virtual memory space
// which is nonexistent or protected. See https://docs.docker.com/config/containers/runmetrics
// for more info.

// CUMULATIVE(memory.stats.pgmajfault): Number of times that a process of the cgroup triggered
// a major page fault. Page faults occur when a process accesses part of its virtual memory space
// which is nonexistent or protected. See https://docs.docker.com/config/containers/runmetrics
// for more info.

// CUMULATIVE(memory.stats.pgpgin): Number of charging events to the memory cgroup. Charging events
// happen each time a page is accounted as either mapped anon page(RSS) or cache page to the cgroup.

// CUMULATIVE(memory.stats.pgpgout): Number of uncharging events to the memory cgroup. Uncharging events
// happen each time a page is unaccounted from the cgroup.

// GAUGE(memory.stats.rss): The amount of memory that doesn’t correspond to anything
// on disk: stacks, heaps, and anonymous memory maps.

// GAUGE(memory.stats.rss_huge): Amount of memory due to anonymous transparent hugepages.

// GAUGE(memory.stats.total_active_anon): Total amount of memory that has been identified as active by
// the kernel. Anonymous memory is memory that is not linked to disk pages.

// GAUGE(memory.stats.total_active_file): Total amount of active file cache memory.
// Cache memory = active_file + inactive_file + tmpfs

// GAUGE(memory.stats.total_cache): Total amount of memory used by the processes of this control group
// that can be associated with a block on a block device. Also accounts for memory used by
// tmpfs.

// GAUGE(memory.stats.total_dirty): Total amount of memory waiting to get written to disk

// GAUGE(memory.stats.total_inactive_anon): Total amount of memory that has been identified as inactive by
// the kernel. Anonymous memory is memory that is not linked to disk pages.

// GAUGE(memory.stats.total_inactive_file): Total amount of inactive file cache memory.
// Cache memory = active_file + inactive_file + tmpfs

// GAUGE(memory.stats.total_mapped_file): Total amount of memory mapped by the processes in
// the control group. It doesn’t give you information about how much memory is used;
// it rather tells you how it is used.

// CUMULATIVE(memory.stats.total_pgfault): Total number of page faults

// CUMULATIVE(memory.stats.total_pgmajfault): Total number of major page faults

// CUMULATIVE(memory.stats.total_pgpgin): Total number of charging events

// CUMULATIVE(memory.stats.total_pgpgout): Total number of uncharging events

// GAUGE(memory.stats.total_rss): Total amount of memory that doesn’t correspond to anything
// on disk: stacks, heaps, and anonymous memory maps.

// GAUGE(memory.stats.total_rss_huge): Total amount of memory due to anonymous transparent hugepages.

// GAUGE(memory.stats.total_unevictable): Total amount of memory that can not be reclaimed

// GAUGE(memory.stats.total_writeback): Total amount of memory from file/anon cache that
// are queued for syncing to the disk

// GAUGE(memory.stats.unevictable): The amount of memory that cannot be reclaimed.

// GAUGE(memory.stats.writeback): The amount of memory from file/anon cache that are queued
// for syncing to the disk

// GAUGE(memory.usage.limit): Memory usage limit of the container, in bytes
// CUSTOM(memory.usage.limit): false

// GAUGE(memory.usage.max): Maximum measured memory usage of the container, in bytes

// GAUGE(memory.usage.total): Bytes of memory used by the container
// CUSTOM(memory.usage.total): false

// CUMULATIVE(network.usage.rx_bytes): Bytes received by the container via its network interface
// CUSTOM(network.usage.rx_bytes): false

// CUMULATIVE(network.usage.rx_dropped): Number of inbound network packets dropped by the container

// CUMULATIVE(network.usage.rx_errors): Errors receiving network packets

// CUMULATIVE(network.usage.rx_packets): Network packets received by the container via its network interface

// CUMULATIVE(network.usage.tx_bytes): Bytes sent by the container via its network interface
// CUSTOM(network.usage.tx_bytes): false

// CUMULATIVE(network.usage.tx_dropped): Number of outbound network packets dropped by the container

// CUMULATIVE(network.usage.tx_errors): Errors sending network packets

// CUMULATIVE(network.usage.tx_packets): Network packets sent by the container via its network interface
