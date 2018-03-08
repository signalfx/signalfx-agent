package rabbitmq

// COUNTER(counter.channel.message_stats.ack): The number of acknowledged
// messages

// COUNTER(counter.channel.message_stats.confirm): Count of messages confirmed.

// COUNTER(counter.channel.message_stats.deliver): Count of messages delivered
// in acknowledgement mode to consumers.

// COUNTER(counter.channel.message_stats.deliver_get): Count of all messages
// delivered on the channel

// COUNTER(counter.channel.message_stats.publish): Count of messages published.

// COUNTER(counter.connection.channel_max): The maximum number of channels on
// the connection

// COUNTER(counter.connection.recv_cnt): Number of packets received on the
// connection

// COUNTER(counter.connection.recv_oct): Number of octets received on the
// connection

// COUNTER(counter.connection.send_cnt): Number of packets sent by the
// connection

// COUNTER(counter.connection.send_oct): Number of octets sent by the
// connection

// COUNTER(counter.exchange.message_stats.confirm): Count of messages
// confirmed.

// COUNTER(counter.exchange.message_stats.publish_in): Count of messages
// published "in" to an exchange, i.e. not taking account of routing.

// COUNTER(counter.exchange.message_stats.publish_out): Count of messages
// published "out" of an exchange, i.e. taking account of routing.

// COUNTER(counter.node.io_read_bytes): Total number of bytes read from disk by
// the persister.

// COUNTER(counter.node.io_read_count): Total number of read operations by the
// persister.

// COUNTER(counter.node.mnesia_disk_tx_count): Number of Mnesia transactions
// which have been performed that required writes to disk.

// COUNTER(counter.node.mnesia_ram_tx_count): Number of Mnesia transactions
// which have been performed that did not require writes to disk.

// COUNTER(counter.queue.disk_reads): Total number of times messages have been
// read from disk by this queue since it started.

// COUNTER(counter.queue.disk_writes): Total number of times messages have been
// written to disk by this queue since it started.

// COUNTER(counter.queue.message_stats.ack): Number of acknowledged messages
// processed by the queue

// COUNTER(counter.queue.message_stats.deliver): Count of messages delivered in
// acknowledgement mode to consumers.

// COUNTER(counter.queue.message_stats.deliver_get): Count of all messages
// delivered on the queue

// COUNTER(counter.queue.message_stats.publish): Count of messages published.

// GAUGE(gauge.channel.connection_details.peer_port): The peer port number of
// the channel

// GAUGE(gauge.channel.consumer_count): The number of consumers the channel has

// GAUGE(gauge.channel.global_prefetch_count): QoS prefetch limit for the
// entire channel, 0 if unlimited.

// GAUGE(gauge.channel.message_stats.ack_details.rate): How much the channel
// message ack count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.channel.message_stats.confirm_details.rate): How much the
// channel message confirm count has changed per second in the most recent
// sampling interval.

// GAUGE(gauge.channel.message_stats.deliver_details.rate): How much the
// channel deliver count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.channel.message_stats.deliver_get_details.rate): How much the
// channel message count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.channel.message_stats.publish_details.rate): How much the
// channel message publish count has changed per second in the most recent
// sampling interval.

// GAUGE(gauge.channel.messages_unacknowledged): Number of messages delivered
// via this channel but not yet acknowledged.

// GAUGE(gauge.channel.messages_uncommitted): Number of messages received in an
// as yet uncommitted transaction.

// GAUGE(gauge.channel.messages_unconfirmed): Number of published messages not
// yet confirmed. On channels not in confirm mode, this remains 0.

// GAUGE(gauge.channel.number): The number of the channel, which uniquely
// identifies it within a connection.

// GAUGE(gauge.channel.prefetch_count): QoS prefetch limit for new consumers, 0
// if unlimited.

// GAUGE(gauge.connection.channels): The current number of channels on the
// connection

// GAUGE(gauge.connection.connected_at): The integer timestamp of the most
// recent time the connection was established

// GAUGE(gauge.connection.frame_max): Maximum permissible size of a frame (in
// bytes) to negotiate with clients.

// GAUGE(gauge.connection.peer_port): The peer port of the connection

// GAUGE(gauge.connection.port): The port the connection is established on

// GAUGE(gauge.connection.recv_oct_details.rate): How much the connection's
// octets received count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.connection.send_oct_details.rate): How much the connection's
// octets sent count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.connection.send_pend): The number of messages in the send queue
// of the connection

// GAUGE(gauge.connection.timeout): The current timeout setting (in seconds) of
// the connection

// GAUGE(gauge.exchange.message_stats.confirm_details.rate): How much the
// message confirm count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.exchange.message_stats.publish_in_details.rate): How much the
// exchange publish-in count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.exchange.message_stats.publish_out_details.rate): How much the
// exchange publish-out count has changed per second in the most recent
// sampling interval.

// GAUGE(gauge.node.disk_free): Disk free space (in bytes) on the node

// GAUGE(gauge.node.disk_free_details.rate): How much the disk free space has
// changed per second in the most recent sampling interval.

// GAUGE(gauge.node.disk_free_limit): Point (in bytes) at which the disk alarm
// will go off.

// GAUGE(gauge.node.fd_total): Total number of file descriptors available.

// GAUGE(gauge.node.fd_used): Number of used file descriptors.

// GAUGE(gauge.node.fd_used_details.rate): How much the number of used file
// descriptors has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.io_read_avg_time): Average wall time (milliseconds) for
// each disk read operation in the last statistics interval.

// GAUGE(gauge.node.io_read_avg_time_details.rate): How much the I/O read
// average time has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.io_read_bytes_details.rate): How much the number of bytes
// read from disk has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.io_read_count_details.rate): How much the number of read
// operations has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.io_sync_avg_time): Average wall time (milliseconds) for
// each fsync() operation in the last statistics interval.

// GAUGE(gauge.node.io_sync_avg_time_details.rate): How much the average I/O
// sync time has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.io_write_avg_time): Average wall time (milliseconds) for
// each disk write operation in the last statistics interval.

// GAUGE(gauge.node.io_write_avg_time_details.rate): How much the I/O write
// time has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.mem_limit): Point (in bytes) at which the memory alarm will
// go off.

// GAUGE(gauge.node.mem_used): Memory used in bytes.

// GAUGE(gauge.node.mem_used_details.rate): How much the count has changed per
// second in the most recent sampling interval.

// GAUGE(gauge.node.mnesia_disk_tx_count_details.rate): How much the Mnesia
// disk transaction count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.node.mnesia_ram_tx_count_details.rate): How much the RAM-only
// Mnesia transaction count has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.node.net_ticktime): Current kernel net_ticktime setting for the
// node.

// GAUGE(gauge.node.proc_total): The maximum number of Erlang processes that
// can run in an Erlang VM.

// GAUGE(gauge.node.proc_used): Number of Erlang processes currently running in
// use.

// GAUGE(gauge.node.proc_used_details.rate): How much the number of erlang
// processes in use has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.node.processors): Number of cores detected and usable by Erlang.

// GAUGE(gauge.node.run_queue): Average number of Erlang processes waiting to
// run.

// GAUGE(gauge.node.sockets_total): Number of file descriptors available for
// use as sockets.

// GAUGE(gauge.node.sockets_used): Number of file descriptors used as sockets.

// GAUGE(gauge.node.sockets_used_details.rate): How much the number of sockets
// used has changed per second in the most recent sampling interval.

// GAUGE(gauge.node.uptime): Time since the Erlang VM started, in milliseconds.

// GAUGE(gauge.queue.backing_queue_status.avg_ack_egress_rate): Rate at which
// unacknowledged message records leave RAM, e.g. because acks arrive or
// unacked messages are paged out

// GAUGE(gauge.queue.backing_queue_status.avg_ack_ingress_rate): Rate at which
// unacknowledged message records enter RAM, e.g. because messages are
// delivered requiring acknowledgement

// GAUGE(gauge.queue.backing_queue_status.avg_egress_rate): Average egress
// (outbound) rate, not including messages that are sent straight through to
// auto-acking consumers.

// GAUGE(gauge.queue.backing_queue_status.avg_ingress_rate): Average ingress
// (inbound) rate, not including messages that are sent straight through to
// auto-acking consumers.

// GAUGE(gauge.queue.backing_queue_status.len): Total backing queue length, in
// messages

// GAUGE(gauge.queue.backing_queue_status.next_seq_id): The next sequence ID to
// be used in the backing queue

// GAUGE(gauge.queue.backing_queue_status.q1): Number of messages in backing
// queue q1

// GAUGE(gauge.queue.backing_queue_status.q2): Number of messages in backing
// queue q2

// GAUGE(gauge.queue.backing_queue_status.q3): Number of messages in backing
// queue q3

// GAUGE(gauge.queue.backing_queue_status.q4): Number of messages in backing
// queue q4

// GAUGE(gauge.queue.consumer_utilisation): Fraction of the time (between 0.0
// and 1.0) that the queue is able to immediately deliver messages to
// consumers.

// GAUGE(gauge.queue.consumers): Number of consumers of the queue

// GAUGE(gauge.queue.memory): Bytes of memory consumed by the Erlang process
// associated with the queue, including stack, heap and internal structures.

// GAUGE(gauge.queue.message_bytes): Sum of the size of all message bodies in
// the queue. This does not include the message properties (including headers)
// or any overhead.

// GAUGE(gauge.queue.message_bytes_persistent): Total number of persistent
// messages in the queue (will always be 0 for transient queues).

// GAUGE(gauge.queue.message_bytes_ram): Like message_bytes but counting only
// those messages which are in RAM.

// GAUGE(gauge.queue.message_bytes_ready): Like message_bytes but counting only
// those messages ready to be delivered to clients.

// GAUGE(gauge.queue.message_bytes_unacknowledged): Like message_bytes but
// counting only those messages delivered to clients but not yet acknowledged.

// GAUGE(gauge.queue.message_stats.ack_details.rate): How much the number of
// acknowledged messages has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.queue.message_stats.deliver_details.rate): How much the count of
// messages delivered has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.queue.message_stats.deliver_get_details.rate): How much the
// count of all messages delivered has changed per second in the most recent
// sampling interval.

// GAUGE(gauge.queue.message_stats.publish_details.rate): How much the count of
// messages published has changed per second in the most recent sampling
// interval.

// GAUGE(gauge.queue.messages): Sum of ready and unacknowledged messages (queue
// depth).

// GAUGE(gauge.queue.messages_details.rate): How much the queue depth has
// changed per second in the most recent sampling interval.

// GAUGE(gauge.queue.messages_persistent): Total number of persistent messages
// in the queue (will always be 0 for transient queues).

// GAUGE(gauge.queue.messages_ram): Total number of messages which are resident
// in RAM.

// GAUGE(gauge.queue.messages_ready): Number of messages ready to be delivered
// to clients.

// GAUGE(gauge.queue.messages_ready_details.rate): How much the count of
// messages ready has changed per second in the most recent sampling interval.

// GAUGE(gauge.queue.messages_ready_ram): Number of messages from
// messages_ready which are resident in RAM.

// GAUGE(gauge.queue.messages_unacknowledged): Number of messages delivered to
// clients but not yet acknowledged.

// GAUGE(gauge.queue.messages_unacknowledged_details.rate): How much the count
// of unacknowledged messages has changed per second in the most recent
// sampling interval.

// GAUGE(gauge.queue.messages_unacknowledged_ram): Number of messages from
// messages_unacknowledged which are resident in RAM.
