package kafkaproducer

var defaultMBeanYAML = `
response-rate:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.response-rate"
    type: "gauge"
    table: false
    attribute: "response-rate"

request-rate:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.request-rate"
    type: "gauge"
    table: false
    attribute: "request-rate"

request-latency-avg:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.request-latency-avg"
    type: "gauge"
    table: false
    attribute: "request-latency-avg"

outgoing-byte-rate:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.outgoing-byte-rate"
    type: "gauge"
    table: false
    attribute: "outgoing-byte-rate"

io-time-ns-avg:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.io-time-ns-avg"
    type: "gauge"
    table: false
    attribute: "io-time-ns-avg"

io-wait-time-ns-avg:
  objectName: "kafka.producer:client-id=*,type=producer-metrics"
  instancePrefix: "all"
  dimensions:
  - client-id
  values:
  - instancePrefix: "kafka.producer.io-wait-time-ns-avg"
    type: "gauge"
    table: false
    attribute: "io-wait-time-ns-avg"
`
