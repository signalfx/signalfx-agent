package activemq

// COUNTER(counter.amq.TotalConnectionsCount): Total connections count per
// broker

// GAUGE(gauge.amq.TotalConsumerCount): Total number of consumers subscribed to
// destinations on the broker

// GAUGE(gauge.amq.TotalDequeueCount): Total number of messages that have been
// acknowledged from the broker.

// GAUGE(gauge.amq.TotalEnqueueCount): Total number of messages that have been
// sent to the broker.

// GAUGE(gauge.amq.TotalMessageCount): Total number of unacknowledged messages
// on the broker

// GAUGE(gauge.amq.TotalProducerCount): Total number of message producers
// active on destinations on the broker

// GAUGE(gauge.amq.queue.AverageBlockedTime): Average time (ms) that messages
// have spent blocked by Flow Control.

// GAUGE(gauge.amq.queue.AverageEnqueueTime): Average time (ms) that messages
// have been held at this destination

// GAUGE(gauge.amq.queue.AverageMessageSize): Average size of messages in this
// queue, in bytes.

// GAUGE(gauge.amq.queue.BlockedSends): Number of messages blocked by Flow
// Control.

// GAUGE(gauge.amq.queue.ConsumerCount): Number of consumers subscribed to this
// queue.

// GAUGE(gauge.amq.queue.DequeueCount): Number of messages that have been
// acknowledged and removed from the queue.

// GAUGE(gauge.amq.queue.EnqueueCount): Number of messages that have been sent
// to the queue.

// GAUGE(gauge.amq.queue.ExpiredCount): Number of messages that have expired
// from the queue.

// GAUGE(gauge.amq.queue.ForwardCount): Number of messages that have been
// forwarded from this queue to a networked broker.

// GAUGE(gauge.amq.queue.InFlightCount): The number of messages that have been
// dispatched to consumers, but not acknowledged.

// GAUGE(gauge.amq.queue.ProducerCount): Number of producers publishing to this
// queue

// GAUGE(gauge.amq.queue.QueueSize): The number of messages in the queue that
// have yet to be consumed.

// GAUGE(gauge.amq.queue.TotalBlockedTime): The total time (ms) that messages
// have spent blocked by Flow Control.

// GAUGE(gauge.amq.topic.AverageBlockedTime): Average time (ms) that messages
// have been blocked by Flow Control.

// GAUGE(gauge.amq.topic.AverageEnqueueTime): Average time (ms) that messages
// have been held at this destination.

// GAUGE(gauge.amq.topic.AverageMessageSize): Average size of messages on this
// topic, in bytes.

// GAUGE(gauge.amq.topic.BlockedSends): Number of messages blocked by Flow
// Control

// GAUGE(gauge.amq.topic.ConsumerCount): The number of consumers subscribed to
// this topic

// GAUGE(gauge.amq.topic.DequeueCount): Number of messages that have been
// acknowledged and removed from the topic.

// GAUGE(gauge.amq.topic.EnqueueCount): The number of messages that have been
// sent to the topic.

// GAUGE(gauge.amq.topic.ExpiredCount): The number of messages that have
// expired from this topic.

// GAUGE(gauge.amq.topic.ForwardCount): The number of messages that have been
// forwarded from this topic to a networked broker.

// GAUGE(gauge.amq.topic.InFlightCount): The number of messages that have been
// dispatched to consumers, but have not yet been acknowledged.

// GAUGE(gauge.amq.topic.ProducerCount): Number of producers publishing to this
// topic.

// GAUGE(gauge.amq.topic.QueueSize): Number of messages in the topic that have
// yet to be consumed.

// GAUGE(gauge.amq.topic.TotalBlockedTime): The total time (ms) that messages
// have spent blocked by Flow Control
