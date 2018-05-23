package couchbase

// GAUGE(gauge.bucket.basic.dataUsed): Size of user data within buckets of the specified state that are resident in RAM (%)

// GAUGE(gauge.bucket.basic.diskFetches): Number of disk fetches

// GAUGE(gauge.bucket.basic.diskUsed): Amount of disk used (bytes)

// GAUGE(gauge.bucket.basic.itemCount): Number of items associated with the bucket

// GAUGE(gauge.bucket.basic.memUsed): Amount of memory used by the bucket (bytes)

// GAUGE(gauge.bucket.basic.opsPerSec): Number of operations per second

// GAUGE(gauge.bucket.basic.quotaPercentUsed): Percentage of RAM used (for active objects) against the configure bucket size (%)

// GAUGE(gauge.bucket.op.cmd_get): requested objects

// GAUGE(gauge.bucket.op.couch_docs_fragmentation): Percent fragmentation of documents in this bucket.

// GAUGE(gauge.bucket.op.couch_views_ops): view operations per second

// GAUGE(gauge.bucket.op.curr_connections): open connection per bucket

// GAUGE(gauge.bucket.op.curr_items): total number of stored items per bucket

// GAUGE(gauge.bucket.op.disk_write_queue): number of items waiting to be written to disk

// GAUGE(gauge.bucket.op.ep_bg_fetched): number of items fetched from disk

// GAUGE(gauge.bucket.op.ep_cache_miss_rate): ratio of requested objects found in cache vs retrieved from disk

// GAUGE(gauge.bucket.op.ep_diskqueue_drain): items removed from disk queue

// GAUGE(gauge.bucket.op.ep_diskqueue_fill): enqueued items on disk queue

// GAUGE(gauge.bucket.op.ep_mem_high_wat): memory high water mark - point at which active objects begin to be ejected from bucket

// GAUGE(gauge.bucket.op.ep_mem_low_wat): memory low water mark

// GAUGE(gauge.bucket.op.ep_num_value_ejects): number of objects ejected out of the bucket

// GAUGE(gauge.bucket.op.ep_oom_errors): request rejected - bucket is at quota, panic

// GAUGE(gauge.bucket.op.ep_queue_size): number of items queued for storage

// GAUGE(gauge.bucket.op.ep_tmp_oom_errors): request rejected - couchbase is making room by ejecting objects, try again later

// GAUGE(gauge.bucket.op.mem_used): memory used

// GAUGE(gauge.bucket.op.ops): total of gets, sets, increment and decrement

// GAUGE(gauge.bucket.op.vb_active_resident_items_ratio): ratio of items kept in memory vs stored on disk

// GAUGE(gauge.bucket.quota.ram): Amount of RAM used by the bucket (bytes).

// GAUGE(gauge.bucket.quota.rawRAM): Amount of raw RAM used by the bucket (bytes).

// GAUGE(gauge.nodes.cmd_get): Number of get commands

// GAUGE(gauge.nodes.couch_docs_actual_disk_size): Amount of disk space used by Couch docs.(bytes)

// GAUGE(gauge.nodes.couch_docs_data_size): Data size of couch documents associated with a node (bytes)

// GAUGE(gauge.nodes.couch_spatial_data_size): Size of object data for spatial views (bytes)

// GAUGE(gauge.nodes.couch_spatial_disk_size): Amount of disk space occupied by spatial views, in bytes.

// GAUGE(gauge.nodes.couch_views_actual_disk_size): Amount of disk space occupied by Couch views (bytes).

// GAUGE(gauge.nodes.couch_views_data_size): Size of object data for Couch views (bytes).

// GAUGE(gauge.nodes.curr_items): Number of current items

// GAUGE(gauge.nodes.curr_items_tot): Total number of items associated with node

// GAUGE(gauge.nodes.ep_bg_fetched): Number of disk fetches performed since server was started

// GAUGE(gauge.nodes.get_hits): Number of get hits

// GAUGE(gauge.nodes.mcdMemoryAllocated): Amount of memcached memory allocated (bytes).

// GAUGE(gauge.nodes.mcdMemoryReserved): Amount of memcached memory reserved (bytes).

// GAUGE(gauge.nodes.mem_used): Memory used by the node (bytes)

// GAUGE(gauge.nodes.memoryFree): Amount of memory free for the node (bytes).

// GAUGE(gauge.nodes.memoryTotal): Total memory available to the node (bytes).

// GAUGE(gauge.nodes.ops): Number of operations performed on Couchbase

// GAUGE(gauge.nodes.system.cpu_utilization_rate): The CPU utilization rate (%)

// GAUGE(gauge.nodes.system.mem_free): Free memory available to the node (bytes)

// GAUGE(gauge.nodes.system.mem_total): Total memory available to the node (bytes)

// GAUGE(gauge.nodes.system.swap_total): Total swap size allocated (bytes)

// GAUGE(gauge.nodes.system.swap_used): Amount of swap space used (bytes)

// GAUGE(gauge.nodes.vb_replica_curr_items): Number of items/documents that are replicas

// GAUGE(gauge.storage.hdd.free): Free harddrive space in the cluster (bytes)

// GAUGE(gauge.storage.hdd.quotaTotal): Harddrive quota total for the cluster (bytes)

// GAUGE(gauge.storage.hdd.total): Total harddrive space available to cluster (bytes)

// GAUGE(gauge.storage.hdd.used): Harddrive space used by the cluster (bytes)

// GAUGE(gauge.storage.hdd.usedByData): Harddrive use by the data in the cluster(bytes)

// GAUGE(gauge.storage.ram.quotaTotal): Ram quota total for the cluster (bytes)

// GAUGE(gauge.storage.ram.quotaTotalPerNode): Ram quota total per node (bytes)

// GAUGE(gauge.storage.ram.quotaUsed): Ram quota used by the cluster (bytes)

// GAUGE(gauge.storage.ram.quotaUsedPerNode): Ram quota used per node (bytes)

// GAUGE(gauge.storage.ram.total): Total ram available to cluster (bytes)

// GAUGE(gauge.storage.ram.used): Ram used by the cluster (bytes)

// GAUGE(gauge.storage.ram.usedByData): Ram used by the data in the cluster (bytes)
