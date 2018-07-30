package kafka

var defaultMBeanYAML = `
kafka-all-messages:
  objectName: "kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec"
  instancePrefix: "all"
  values:
  - instancePrefix: "kafka-messages-in"
    type: "counter"
    table: false
    attribute: "Count"

kafka-all-bytes-in:
  objectName: "kafka.server:type=BrokerTopicMetrics,name=BytesInPerSec"
  instancePrefix: "all"
  values:
  - instancePrefix: "kafka-bytes-in"
    type: "counter"
    table: false
    attribute: "Count"

kafka-all-bytes-out:
  objectName: "kafka.server:type=BrokerTopicMetrics,name=BytesOutPerSec"
  instancePrefix: "all"
  values:
  - instancePrefix: "kafka-bytes-out"
    type: "counter"
    table: false
    attribute: "Count"

kafka-active-controllers:
  objectName: "kafka.controller:type=KafkaController,name=ActiveControllerCount"
  values:
  - type: "gauge"
    table: false
    attribute: "Value"
    instancePrefix: "kafka-active-controllers"

kafka-offline-partitions-count:
  objectName: "kafka.controller:type=KafkaController,name=OfflinePartitionsCount"
  values:
  - type: "gauge"
    table: false
    attribute: "Value"
    instancePrefix: "kafka-offline-partitions-count"

kafka-underreplicated-partitions:
  objectName: "kafka.server:type=ReplicaManager,name=UnderReplicatedPartitions"
  values:
  - type: "gauge"
    table: false
    attribute: "Value"
    instancePrefix: "kafka-underreplicated-partitions"

kafka-request-queue:
  objectName: "kafka.network:type=RequestChannel,name=RequestQueueSize"
  values:
  - type: "gauge"
    table: false
    attribute: "Value"
    instancePrefix: "kafka-request-queue"

kafka.fetch-consumer.total-time:
  objectName: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=FetchConsumer"
  values:
  - type: "counter"
    table: false
    attribute: "Count"
    instancePrefix: "kafka.fetch-consumer.total-time.count"
  - type: "gauge"
    table: false
    attribute: "50thPercentile"
    instancePrefix: "kafka.fetch-consumer.total-time.median"
  - type: "gauge"
    table: false
    attribute: "99thPercentile"
    instancePrefix: "kafka.fetch-consumer.total-time.99th"

kafka.fetch-follower.total-time:
  objectName: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=FetchFollower"
  values:
  - type: "counter"
    table: false
    attribute: "Count"
    instancePrefix: "kafka.fetch-follower.total-time.count"
  - type: "gauge"
    table: false
    attribute: "50thPercentile"
    instancePrefix: "kafka.fetch-follower.total-time.median"
  - type: "gauge"
    table: false
    attribute: "99thPercentile"
    instancePrefix: "kafka.fetch-follower.total-time.99th"

kafka.produce.total-time:
  objectName: "kafka.network:type=RequestMetrics,name=TotalTimeMs,request=Produce"
  values:
  - type: "counter"
    table: false
    attribute: "Count"
    instancePrefix: "kafka.produce.total-time.count"
  - type: "gauge"
    table: false
    attribute: "50thPercentile"
    instancePrefix: "kafka.produce.total-time.median"
  - type: "gauge"
    table: false
    attribute: "99thPercentile"
    instancePrefix: "kafka.fetchproducetotal-time.99th"
`
