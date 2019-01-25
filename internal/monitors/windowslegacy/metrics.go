package windowslegacy

// GAUGE(processor.pct_processor_time): Percentage of elapsed time the processor
// spends executing non-idle threads.

// GAUGE(processor.pct_privileged_time): Percentage of elapsed time the
// processor spends on privileged tasks.

// GAUGE(processor.pct_user_time): Percentage of elapsed time the processor
// spends executing user threads.

// GAUGE(processor.interrupts_sec): Rate of processor interrupts per second.

// GAUGE(system.processor_queue_length): Number of threads waiting for CPU
// cycles, where < 12 per CPU is good/fair, < 8 is better, < 4 is best

// GAUGE(system.system_calls_sec): The number of system calls being serviced by
// the CPU per second.

// GAUGE(system.context_switches_sec): Number of execution contexts switched in
// the last second, where >6000 is poor, <3000 is good, and <1500 is excellent.

// GAUGE(memory.available_mbytes): Unused physical memory (not page file).

// GAUGE(memory.pages_input_sec): Reads from hard disk per second to resolve
// hard pages.

// GAUGE(paging_file.pct_usage): Amount of Page File in use, which indicates the
// server is substituting disk space for memory.

// GAUGE(paging_file.pct_usage_peak): Highest %Usage metric since the last time
// the server was restarted.

// GAUGE(physicaldisk.avg_disk_sec_write): The average time, in milliseconds, of
// each write to disk.

// GAUGE(physicaldisk.avg_disk_sec_read): The average time, in milliseconds, of
// each read from disk.

// GAUGE(physicaldisk.avg_disk_sec_transfer): The average time in milliseconds
// spent transfering data on disk.

// GAUGE(logicaldisk.disk_read_bytes_sec): The number of bytes read from
// disk per second.

// GAUGE(logicaldisk.disk_write_bytes_sec): The number of bytes written
// to disk per second.

// GAUGE(logicaldisk.disk_transfers_sec): The number of transfers per
// second.

// GAUGE(logicaldisk.disk_reads_sec): The number of read operations per second.
// GAUGE(logicaldisk.disk_writes_sec): The number of write operations per
// second.

// GAUGE(logicaldisk.free_megabytes): The number of available megabytes.

// GAUGE(logicaldisk.pct_free_space): The percentage of free disk space
// available.

// GAUGE(network_interface.bytes_total_sec): The number of bytes sent and
// received over a specific network adapter, including framing characters.

// GAUGE(network_interface.bytes_received_sec): Bytes Received/sec is the rate
// at which bytes are received over each network adapter, including framing
// characters.

// GAUGE(network_interface.bytes_sent_sec): Bytes Sent/sec is the rate at which
// bytes are sent over each network adapter, including framing characters.

// GAUGE(network_interface.current_bandwidth): Current Bandwidth is an estimate
// of the current bandwidth of the network interface in bits per second (BPS).

// GAUGE(network_interface.packets_received_sec): Tracking the packets received
// over time can give you a good indication of the typical use of the system's
// network.

// GAUGE(network_interface.packets_sent_sec): The number of packets sent
// per second.

// GAUGE(network_interface.packets_received_errors): The number of packets
// received that encountered an error.

// GAUGE(network_interface.packets_outbound_errors): The number of packets sent
// that encountered an error.

// GAUGE(network_interface.received_discarded): The number of received packets
// discarded.

// GAUGE(network_interface.outbound_discarded): The number of outbound packets
// discarded
