package genericjmx

// MBeanMap is a map from the service name to the mbean definitions that this
// service has
type MBeanMap map[string]MBean

// MergeWith combines the current MBeanMap with the one given as an
// argument and returns a new map with values from both maps.
func (m MBeanMap) MergeWith(m2 MBeanMap) MBeanMap {
	out := MBeanMap{}
	for k, v := range m {
		out[k] = v
	}
	for k, v := range m2 {
		out[k] = v
	}
	return out
}

func (m MBeanMap) MBeanNames() []string {
	names := make([]string, 0)
	for n := range m {
		names = append(names, n)
	}
	return names
}

// DefaultMBeans are basic JVM memory and threading metrics that are common to
// all JMX applications
var DefaultMBeans MBeanMap

const defaultMBeanYAML = `
garbage_collector:
  objectName: "java.lang:type=GarbageCollector,*"
  instancePrefix: "gc-"
  instanceFrom:
  - "name"
  values:
  - type: "invocations"
    table: false
    attribute: "CollectionCount"
  - type: "total_time_in_ms"
    instancePrefix: "collection_time"
    table: false
    attribute: "CollectionTime"

memory-heap:
  objectName: java.lang:type=Memory
  instancePrefix: memory-heap
  values:
  - type: jmx_memory
    table: true
    attribute: HeapMemoryUsage

memory-nonheap:
  objectName: java.lang:type=Memory
  instancePrefix: memory-nonheap
  values:
  - type: jmx_memory
    table: true
    attribute: NonHeapMemoryUsage

memory_pool:
  objectName: java.lang:type=MemoryPool,*
  instancePrefix: memory_pool-
  instanceFrom:
  - name
  values:
  - type: jmx_memory
    table: true
    attribute: Usage

threading:
  objectName: java.lang:type=Threading
  values:
  - type: gauge
    table: false
    instancePrefix: jvm.threads.count
    attribute: ThreadCount
`

// MBean represents the <MBean> config object in the collectd config for
// generic jmx.
type MBean struct {
	ObjectName     string   `yaml:"objectName"`
	InstancePrefix string   `yaml:"instancePrefix"`
	InstanceFrom   []string `yaml:"instanceFrom"`
	Values         []struct {
		Type           string `yaml:"type"`
		Table          bool   `yaml:"table"`
		InstancePrefix string `yaml:"instancePrefix"`
		InstanceFrom   string `yaml:"instanceFrom"`
		Attribute      string `yaml:"attribute"`
	} `yaml:"values"`
}
