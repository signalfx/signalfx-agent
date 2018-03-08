package memcached

// GAUGE(df.cache.free): Unused storage bytes

// GAUGE(df.cache.used): Current number of bytes used to store items

// CUMULATIVE(memcached_command.flush): Number of flush requests

// CUMULATIVE(memcached_command.get): Number of retrieval requests

// CUMULATIVE(memcached_command.set): Number of storage requests

// CUMULATIVE(memcached_command.touch): Number of touch requests

// GAUGE(memcached_connections.current): Current number of open connections

// GAUGE(memcached_connections.listen_disabled): Number of times connection
// limit has been exceeded

// GAUGE(memcached_items.current): Current number of items stored by this
// instance

// CUMULATIVE(memcached_octets.rx): Total network bytes read by this server

// CUMULATIVE(memcached_octets.tx): Total network bytes written by this server

// CUMULATIVE(memcached_ops.decr_hits): Number of successful Decr requests

// CUMULATIVE(memcached_ops.decr_misses): Number of decr requests against
// missing keys

// CUMULATIVE(memcached_ops.evictions): Number of valid items removed from
// cache

// CUMULATIVE(memcached_ops.hits): Number of keys that have been requested and
// found present

// CUMULATIVE(memcached_ops.incr_hits): Number of successful incr requests

// CUMULATIVE(memcached_ops.incr_misses): Number of incr requests against
// missing keys

// CUMULATIVE(memcached_ops.misses): Number of items that have been requested
// and not found

// GAUGE(ps_count.threads): Number of worker threads requested

// CUMULATIVE(ps_cputime.syst): Total system time for this instance

// CUMULATIVE(ps_cputime.user): Total user time for this instance
