// +build linux

package cassandra

var defaultMBeanYAML = `
cassandra-client-read-latency:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Read,name=Latency
  values:
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Read.Latency.50thPercentile
    attribute: 50thPercentile
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Read.Latency.Max
    attribute: Max
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Read.Latency.99thPercentile
    attribute: 99thPercentile
  - type: counter
    instancePrefix: cassandra.ClientRequest.Read.Latency.Count
    attribute: Count


cassandra-client-read-timeouts:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Read,name=Timeouts
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.Read.Timeouts.Count
    attribute: Count


cassandra-client-read-unavailables:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Read,name=Unavailables
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.Read.Unavailables.Count
    attribute: Count


cassandra-client-rangeslice-latency:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=RangeSlice,name=Latency
  values:
  - type: gauge
    instancePrefix: cassandra.ClientRequest.RangeSlice.Latency.50thPercentile
    attribute: 50thPercentile
  - type: gauge
    instancePrefix: cassandra.ClientRequest.RangeSlice.Latency.Max
    attribute: Max
  - type: gauge
    instancePrefix: cassandra.ClientRequest.RangeSlice.Latency.99thPercentile
    attribute: 99thPercentile
  - type: counter
    instancePrefix: cassandra.ClientRequest.RangeSlice.Latency.Count
    attribute: Count


cassandra-client-rangeslice-timeouts:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=RangeSlice,name=Timeouts
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.RangeSlice.Timeouts.Count
    attribute: Count


cassandra-client-rangeslice-unavailables:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=RangeSlice,name=Unavailables
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.RangeSlice.Unavailables.Count
    attribute: Count


cassandra-client-write-latency:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency
  values:
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Write.Latency.50thPercentile
    attribute: 50thPercentile
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Write.Latency.Max
    attribute: Max
  - type: gauge
    instancePrefix: cassandra.ClientRequest.Write.Latency.99thPercentile
    attribute: 99thPercentile
  - type: counter
    instancePrefix: cassandra.ClientRequest.Write.Latency.Count
    attribute: Count


cassandra-client-write-timeouts:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Timeouts
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.Write.Timeouts.Count
    attribute: Count


cassandra-client-write-unavailables:
  objectName: org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Unavailables
  values:
  - type: counter
    instancePrefix: cassandra.ClientRequest.Write.Unavailables.Count
    attribute: Count


cassandra-storage-load:
  objectName: org.apache.cassandra.metrics:type=Storage,name=Load
  values:
  - type: gauge
    instancePrefix: cassandra.Storage.Load.Count
    attribute: Count


cassandra-storage-hints:
  objectName: org.apache.cassandra.metrics:type=Storage,name=TotalHints
  values:
  - type: gauge
    instancePrefix: cassandra.Storage.TotalHints.Count
    attribute: Count


cassandra-storage-hints-in-progress:
  objectName: org.apache.cassandra.metrics:type=Storage,name=TotalHintsInProgress
  values:
  - type: gauge
    instancePrefix: cassandra.Storage.TotalHintsInProgress.Count
    attribute: Count


cassandra-compaction-pending-tasks:
  objectName: org.apache.cassandra.metrics:type=Compaction,name=PendingTasks
  values:
  - type: gauge
    instancePrefix: cassandra.Compaction.PendingTasks.Value
    attribute: Value


cassandra-compaction-total-completed:
  objectName: org.apache.cassandra.metrics:type=Compaction,name=TotalCompactionsCompleted
  values:
  - type: counter
    instancePrefix: cassandra.Compaction.TotalCompactionsCompleted.Count
    attribute: Count
`
