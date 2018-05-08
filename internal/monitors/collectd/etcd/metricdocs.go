package etcd

// COUNTER(counter.etcd.leader.counts.fail): Total number of failed rpc requests to with a follower

// COUNTER(counter.etcd.leader.counts.success): Total number of successful rpc requests to with a follower

// COUNTER(counter.etcd.self.recvappendreq.cnt): Total number of append requests received by a member

// COUNTER(counter.etcd.self.sendappendreq.cnt): Total number of append requests sent by a member

// COUNTER(counter.etcd.store.compareanddelete.fail): Total number of failed compare-and-delete operations

// COUNTER(counter.etcd.store.compareanddelete.success): Total number of successful compare-and-delete operations

// COUNTER(counter.etcd.store.compareandswap.fail): Total number of failed compare-and-swap operations

// COUNTER(counter.etcd.store.compareandswap.success): Total number of successful compare-and-swap operations

// COUNTER(counter.etcd.store.create.fail): Total number of failed create operations

// COUNTER(counter.etcd.store.create.success): Total number of successful create operations

// COUNTER(counter.etcd.store.delete.fail): Total number of failed delete operations

// COUNTER(counter.etcd.store.delete.success): Total number of successful delete operations

// COUNTER(counter.etcd.store.expire.count): Total number of items expired due to TTL

// COUNTER(counter.etcd.store.gets.fail): Total number of failed get operations

// COUNTER(counter.etcd.store.gets.success): Total number of successful get operations

// COUNTER(counter.etcd.store.sets.fail): Total number of failed set operations

// COUNTER(counter.etcd.store.sets.success): Total number of successful set operations

// COUNTER(counter.etcd.store.update.fail): Total number of failed update operations

// COUNTER(counter.etcd.store.update.success): Total number of successful update operations

// GAUGE(gauge.etcd.leader.latency.average): Average latency of a follower with respect to the leader

// GAUGE(gauge.etcd.leader.latency.current): Current latency of a follower with respect to the leader

// GAUGE(gauge.etcd.leader.latency.max): Max latency of a follower with respect to the leader

// GAUGE(gauge.etcd.leader.latency.min): Min latency of a follower with respect to the leader

// GAUGE(gauge.etcd.leader.latency.stddev): Std dev latency of a follower with respect to the leader

// GAUGE(gauge.etcd.self.recvbandwidth.rate): Bandwidth rate of a follower

// GAUGE(gauge.etcd.self.recvpkg.rate): Rate at which a follower receives packages

// GAUGE(gauge.etcd.self.sendbandwidth.rate): Bandwidth rate of a leader

// GAUGE(gauge.etcd.self.sendpkg.rate): Rate at which a leader sends packages

// GAUGE(gauge.etcd.store.watchers): Number of watchers

