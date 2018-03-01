package consul

// GAUGE(consul.dns.stale_queries): Number of times an agent serves a DNS query
// with stale information

// GAUGE(consul.memberlist.msg.suspect): Number of suspect messages received
// per interval

// GAUGE(consul.serf.member.flap): Tracks flapping agents

// GAUGE(gauge.consul.catalog.nodes.total): Number of nodes in the Consul
// datacenter

// GAUGE(gauge.consul.catalog.nodes_by_service): Number of nodes providing a
// given service

// GAUGE(gauge.consul.catalog.services.total): Total number of services
// registered with Consul in the given datacenter

// GAUGE(gauge.consul.catalog.services_by_node): Number of services registered
// with a node

// GAUGE(gauge.consul.consul.dns.domain_query.AGENT.avg): Average time to
// complete a forward DNS query

// GAUGE(gauge.consul.consul.dns.domain_query.AGENT.max): Max time to complete
// a forward DNS query

// GAUGE(gauge.consul.consul.dns.domain_query.AGENT.min): Min time to complete
// a forward DNS query

// GAUGE(gauge.consul.consul.dns.ptr_query.AGENT.avg): Average time to complete
// a Reverse DNS query

// GAUGE(gauge.consul.consul.dns.ptr_query.AGENT.max): Max time to complete a
// Reverse DNS query

// GAUGE(gauge.consul.consul.dns.ptr_query.AGENT.min): Min time to complete a
// Reverse DNS query

// GAUGE(gauge.consul.consul.leader.reconcile.avg): Leader time to reconcile
// the differences between Serf membership and Consul's store

// GAUGE(gauge.consul.health.nodes.critical): Number of nodes for which health
// checks are reporting Critical state

// GAUGE(gauge.consul.health.nodes.passing): Number of nodes for which health
// checks are reporting Passing state

// GAUGE(gauge.consul.health.nodes.warning): Number of nodes for which health
// checks are reporting Warning state

// GAUGE(gauge.consul.health.services.critical): Number of services for which
// health checks are reporting Critical state

// GAUGE(gauge.consul.health.services.passing): Number of services for which
// health checks are reporting Passing state

// GAUGE(gauge.consul.health.services.warning): Number of services for which
// health checks are reporting Warning state

// GAUGE(gauge.consul.is_leader): Metric to map consul server's in leader or
// follower state

// GAUGE(gauge.consul.network.dc.latency.avg): Average network latency between
// 2 datacenters

// GAUGE(gauge.consul.network.dc.latency.max): Maximum network latency between
// 2 datacenters

// GAUGE(gauge.consul.network.dc.latency.min): Minimum network latency between
// 2 datacenters

// GAUGE(gauge.consul.network.node.latency.avg): Average network latency
// between given node and other nodes in the datacenter

// GAUGE(gauge.consul.network.node.latency.max): Minimum network latency
// between given node and other nodes in the datacenter

// GAUGE(gauge.consul.network.node.latency.min): Minimum network latency
// between given node and other nodes in the datacenter

// GAUGE(gauge.consul.peers): Number of Raft peers in Consul datacenter

// GAUGE(gauge.consul.raft.apply): Number of raft transactions

// GAUGE(gauge.consul.raft.commitTime.avg): Average of the time it takes to
// commit an entry on the leader

// GAUGE(gauge.consul.raft.commitTime.max): Max of the time it takes to commit
// an entry on the leader

// GAUGE(gauge.consul.raft.commitTime.min): Minimum of the time it takes to
// commit an entry on the leader

// GAUGE(gauge.consul.raft.leader.dispatchLog.avg): Average of the time it
// takes for the leader to write log entries to disk

// GAUGE(gauge.consul.raft.leader.dispatchLog.max): Maximum of the time it
// takes for the leader to write log entries to disk

// GAUGE(gauge.consul.raft.leader.dispatchLog.min): Minimum of the time it
// takes for the leader to write log entries to disk

// GAUGE(gauge.consul.raft.leader.lastContact.avg): Mean of the time since the
// leader was last able to contact follower nodes

// GAUGE(gauge.consul.raft.leader.lastContact.max): Max of the time since the
// leader was last able to contact follower nodes

// GAUGE(gauge.consul.raft.leader.lastContact.min): Min of the time since the
// leader was last able to contact follower nodes

// GAUGE(gauge.consul.raft.replication.appendEntries.rpc.AGENT.avg): Mean time
// taken to complete the AppendEntries RPC

// GAUGE(gauge.consul.raft.replication.appendEntries.rpc.AGENT.max): Max time
// taken to complete the AppendEntries RPC

// GAUGE(gauge.consul.raft.replication.appendEntries.rpc.AGENT.min): Min time
// taken to complete the AppendEntries RPC

// GAUGE(gauge.consul.raft.state.candidate): Tracks the number of times given
// node enters the candidate state

// GAUGE(gauge.consul.raft.state.leader): Tracks the number of leadership
// transitions per interval

// GAUGE(gauge.consul.runtime.alloc_bytes): Number of bytes allocated to Consul
// process on the node

// GAUGE(gauge.consul.runtime.heap_objects): Number of heap objects allocated
// to Consul

// GAUGE(gauge.consul.runtime.num_goroutines): Number of GO routines run by
// Consul process

// GAUGE(gauge.consul.serf.events): Number of serf events processed

// GAUGE(gauge.consul.serf.member.join): Tracks successful node joins

// GAUGE(gauge.consul.serf.member.left): Tracks successful node leaves

// GAUGE(gauge.consul.serf.queue.Event.avg): Average number of serf events in
// queue yet to be processed

// GAUGE(gauge.consul.serf.queue.Event.max): Maximum number of serf events in
// queue yet to be processed during the interval

// GAUGE(gauge.consul.serf.queue.Event.min): Minimum number of serf events in
// queue yet to be processed during the interval

// GAUGE(gauge.consul.serf.queue.Query.avg): Average number of serf queries in
// queue yet to be processed during the interval

// GAUGE(gauge.consul.serf.queue.Query.max): Maximum number of serf queries in
// queue yet to be processed during the interval

// GAUGE(gauge.consul.serf.queue.Query.min): Minimum number of serf queries in
// queue yet to be processed during the interval

// DIMENSION(consul_mode): Whether this consul instance is running as a server
// or client
// DIMENSION(consul_node): The name of the consul node
// DIMENSION(datacenter): The name of the consul datacenter
