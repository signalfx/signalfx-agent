package mongodb

// GAUGE(counter.backgroundFlushing.flushes): Number of times the database has been flushed

// COUNTER(counter.collection.commandsCount): Number of commands issued for a collection

// COUNTER(counter.collection.commandsTime): Time spent in microseconds processing commands issued for a collection

// COUNTER(counter.collection.getmoreCount): Number of getMore requests issued for a collection

// COUNTER(counter.collection.getmoreTime): Time spent in microseconds processing getMore requests for a collection

// COUNTER(counter.collection.index.accesses.ops): Number of times an index has been used (only on Mongo 3.2+)

// COUNTER(counter.collection.insertCount): Number of inserts issued for a collection

// COUNTER(counter.collection.insertTime): Time spent in microseconds processing insert requests for a collection

// COUNTER(counter.collection.queriesCount): Number of queries issued for a collection

// COUNTER(counter.collection.queriesTime): Time spent in microseconds processing query requests for a collection

// COUNTER(counter.collection.readLockCount): Number of read locks issued for a collection

// COUNTER(counter.collection.readLockTime): Time spent in microseconds processing read locks for a collection

// COUNTER(counter.collection.removeCount): Number of remove requests issued for a collection

// COUNTER(counter.collection.removeTime): Time spent in microseconds processing remove requests for a collection

// COUNTER(counter.collection.totalCount): Total number of operations issued for a collection

// COUNTER(counter.collection.totalTime): Time spent in microseconds processing all operations for a collection

// COUNTER(counter.collection.updateCount): Number of update requests issued for a collection

// COUNTER(counter.collection.updateTime): Time spent in microseconds processing update requests for a collection

// COUNTER(counter.collection.writeLockCount): Number of write locks issued for a collection

// COUNTER(counter.collection.writeLockTime): Time spent in microseconds processing write locks for a collection

// GAUGE(counter.extra_info.page_faults): Mongod page faults

// GAUGE(counter.network.bytesIn): Network bytes received by the database server

// GAUGE(counter.network.bytesOut): Network bytes sent by the database server

// CUMULATIVE(counter.network.numRequests): Requests received by the server

// CUMULATIVE(counter.opcounters.delete): Number of deletes per second

// CUMULATIVE(counter.opcounters.insert): Number of inserts per second

// CUMULATIVE(counter.opcounters.query): Number of queries per second

// CUMULATIVE(counter.opcounters.update): Number of updates per second

// GAUGE(gauge.backgroundFlushing.average_ms): Average time (ms) to write data to disk

// GAUGE(gauge.backgroundFlushing.last_ms): Most recent time (ms) spent writing data to disk

// GAUGE(gauge.collection.avgObjSize): Mean object/document size of a collection

// GAUGE(gauge.collection.count): Number of objects/documents in a collection

// GAUGE(gauge.collection.indexSize): Size of a particular index on a collection

// GAUGE(gauge.collection.max): Maximum number of documents in a capped collection

// GAUGE(gauge.collection.maxSize): Maximum disk usage of a capped collection

// GAUGE(gauge.collection.size): Size of a collection in bytes, not including indexes

// GAUGE(gauge.collection.storageSize): Size of the collection on disk in bytes, never decreases.

// GAUGE(gauge.collections): Number of collections

// GAUGE(gauge.connections.available): Number of available incoming connections

// GAUGE(gauge.connections.current): Number of current client connections

// GAUGE(gauge.dataSize): Total size of data, in bytes

// GAUGE(gauge.extra_info.heap_usage_bytes): Heap size used by the mongod process, in bytes

// GAUGE(gauge.globalLock.activeClients.readers): Number of active client connections performing reads

// GAUGE(gauge.globalLock.activeClients.total): Total number of active client connections

// GAUGE(gauge.globalLock.activeClients.writers): Number of active client connections performing writes

// GAUGE(gauge.globalLock.currentQueue.readers): Read operations currently in queue

// GAUGE(gauge.globalLock.currentQueue.total): Total operations currently in queue

// GAUGE(gauge.globalLock.currentQueue.writers): Write operations currently in queue

// GAUGE(gauge.indexSize): Total size of indexes, in bytes

// GAUGE(gauge.indexes): Number of indexes across all collections

// GAUGE(gauge.mem.mapped): Mongodb mapped memory usage, in MB

// GAUGE(gauge.mem.resident): Mongodb resident memory usage, in MB

// GAUGE(gauge.mem.virtual): Mongodb virtual memory usage, in MB

// GAUGE(gauge.objects): Number of documents across all collections

// GAUGE(gauge.storageSize): Total bytes allocated to collections for document storage

// COUNTER(gauge.uptime): Uptime of this server in milliseconds

// DIMENSION(plugin_instance): Port number of the MongoDB instance

