package kafka_consumer

var defaultMBeanYAML = `
records-lag-max:
  objectName: "kafka.consumer:client-id=*,type=consumer-fetch-manager-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.consumer.records-lag-max"
    type: "gauge"
    table: false
    attribute: "records-lag-max"

bytes-consumed-rate:
  objectName: "kafka.consumer:client-id=*,type=consumer-fetch-manager-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.consumer.bytes-consumed-rate"
    type: "gauge"
    table: false
    attribute: "bytes-consumed-rate"

records-consumed-rate:
  objectName: "kafka.consumer:client-id=*,type=consumer-fetch-manager-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.consumer.records-consumed-rate"
    type: "gauge"
    table: false
    attribute: "records-consumed-rate"

fetch-rate:
  objectName: "kafka.consumer:client-id=*,type=consumer-fetch-manager-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.consumer.fetch-rate"
    type: "gauge"
    table: false
    attribute: "fetch-rate"
`
