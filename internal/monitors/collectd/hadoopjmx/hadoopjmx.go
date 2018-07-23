// +build !windows

package hadoopjmx

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	yaml "gopkg.in/yaml.v2"
)

const monitorType = "collectd/hadoopjmx"

var serviceName = "hadoop"

type nodeType string

const (
	nameNode        nodeType = "nameNode"
	resourceManager          = "resourceManager"
	nodeManager              = "nodeManager"
	dataNode                 = "dataNode"
)

// MONITOR(collectd/hadoopjmx): Collects metrics about a Hadoop cluster using
// using collectd's GenericJMX plugin.
//
// Also see
// https://github.com/signalfx/integrations/tree/master/collectd-hadoop.
//
// >To enable JMX in Hadoop, add the following JVM options to hadoop-env.sh and yarn-env.sh respectively
//
// **hadoop-env.sh:**
// ```
// export HADOOP_NAMENODE_OPTS="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=5677 $HADOOP_NAMENODE_OPTS"
// export HADOOP_DATANODE_OPTS="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=5679 $HADOOP_DATANODE_OPTS"
// ```
//
// **yarn-env.sh:**
// ```
// export YARN_NODEMANAGER_OPTS="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=5678 $YARN_NODEMANAGER_OPTS"
// export YARN_NODEMANAGER_OPTS="-Dcom.sun.management.jmxremote.ssl=false -Dcom.sun.management.jmxremote.authenticate=false -Dcom.sun.management.jmxremote.port=5678 $YARN_NODEMANAGER_OPTS"
// ```
//
// This monitor has a set of built in MBeans configured for:
// - [Name Nodes](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/hadoopjmx/nameNodeMBeans.go)
// - [Resource Manager](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/hadoopjmx/resourceManagerMBeans.go)
// - [Node Manager](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/hadoopjmx/nodeManagerMBeans.go)
// - [Data Nodes](https://github.com/signalfx/signalfx-agent/tree/master/internal/monitors/collectd/hadoopjmx/dataNodeMBeans.go)
//
// Sample YAML configuration:
//
//	Name Node
// ```yaml
// monitors:
// - type: collectd/hadoopjmx
//   host: 127.0.0.1
//   port: 5677
//   nodeType: nameNode
// ```
//
// 	Resource Manager
// ```yaml
// monitors:
// - type: collectd/hadoopjmx
//   host: 127.0.0.1
//   port: 5680
//   nodeType: resourceManager
// ```
//
// 	Node Manager
// ```yaml
// monitors:
// - type: collectd/hadoopjmx
//   host: 127.0.0.1
//   port: 8002
//   nodeType: nodeManager
// ```
//
// 	Data Node
// ```yaml
// monitors:
// - type: collectd/hadoopjmx
//   host: 127.0.0.1
//   port: 5679
//   nodeType: dataNode
// ```
//
// You may also configure the [collectd/hadoop](https://github.com/signalfx/signalfx-agent/tree/master/docs/monitors/collectd/hadoop)
// monitor to collect additional metrics about the hadoop cluster from the REST API
//

// Config is the monitor-specific config with the generic config embedded
type Config struct {
	genericjmx.Config `yaml:",inline"`
	// Hadoop Node Type
	NodeType nodeType `yaml:"nodeType" validate:"required"`
}

// Monitor is the main type that represents the monitor
type Monitor struct {
	*genericjmx.JMXMonitorCore
}

// Configure configures the hadoopjmx monitor and instantiates the generic jmx
// monitor
func (m *Monitor) Configure(conf *Config) error {
	// create the mbean map with the appropriate mbeans for the given node type
	var newMBeans genericjmx.MBeanMap
	switch conf.NodeType {
	case nameNode:
		newMBeans = genericjmx.DefaultMBeans.MergeWith(loadMBeans(defaultNameNodeMBeanYAML))
	case dataNode:
		newMBeans = genericjmx.DefaultMBeans.MergeWith(loadMBeans(defaultDataNodeMBeanYAML))
	case resourceManager:
		newMBeans = genericjmx.DefaultMBeans.MergeWith(loadMBeans(defaultResourceManagerMBeanYAML))
	case nodeManager:
		newMBeans = genericjmx.DefaultMBeans.MergeWith(loadMBeans(defaultNodeManagerMBeanYAML))
	}

	m.JMXMonitorCore.DefaultMBeans = newMBeans

	// invoke the JMXMonitorCore configuration callback
	return m.JMXMonitorCore.Configure(&conf.Config)
}

// loadMBeans validates the mbean yaml and unmarshals the mbean yaml to an MBeanMap
func loadMBeans(mBeanYaml string) genericjmx.MBeanMap {
	var mbeans genericjmx.MBeanMap

	if err := yaml.Unmarshal([]byte(mBeanYaml), &mbeans); err != nil {
		panic("YAML for GenericJMX MBeans is invalid: " + err.Error())
	}

	return mbeans
}

func init() {
	monitors.Register(monitorType, func() interface{} {
		return &Monitor{
			genericjmx.NewJMXMonitorCore(genericjmx.DefaultMBeans, serviceName),
		}
	}, &Config{})
}
