package memory

// GAUGE(memory.buffered): (Linux Only) Bytes of memory used for buffering I/O.

// GAUGE(memory.cached): (Linux Only) Bytes of memory used for disk caching.

// GAUGE(memory.free): (Linux Only) Bytes of memory available for use.

// GAUGE(memory.available): (Windows Only) Bytes of memory available for use.

// GAUGE(memory.slab_recl): (Linux Only) Bytes of memory, used for SLAB-allocation of kernel
// objects, that can be reclaimed.

// GAUGE(memory.slab_unrecl): (Linux Only) Bytes of memory, used for SLAB-allocation of
// kernel objects, that can't be reclaimed.

// GAUGE(memory.used): Bytes of memory in use by the system.

// GAUGE(memory.utilization): Percent of memory in use on this host.
// This metric reports with plugin dimension set to "system-utilization".
