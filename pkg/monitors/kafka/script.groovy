ss = util.queryJMX("kafka.server:type=BrokerTopicMetrics,name=MessagesInPerSec").first()

kafka-all-messages:
  objectName: ""
  instancePrefix: "all"
  values:
  - instancePrefix: "kafka-messages-in"
    type: "counter"
    table: false
    attribute: "Count"
