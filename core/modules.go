package core

// Do an import of all of the built-in observers and monitors until we get a
// proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/neo-agent/monitors/cadvisor"
	_ "github.com/signalfx/neo-agent/monitors/collectd/activemq"
	_ "github.com/signalfx/neo-agent/monitors/collectd/apache"
	_ "github.com/signalfx/neo-agent/monitors/collectd/cassandra"
	_ "github.com/signalfx/neo-agent/monitors/collectd/custom"
	_ "github.com/signalfx/neo-agent/monitors/collectd/docker"
	_ "github.com/signalfx/neo-agent/monitors/collectd/elasticsearch"
	_ "github.com/signalfx/neo-agent/monitors/collectd/healthchecker"
	_ "github.com/signalfx/neo-agent/monitors/collectd/kafka"
	_ "github.com/signalfx/neo-agent/monitors/collectd/marathon"
	_ "github.com/signalfx/neo-agent/monitors/collectd/memcached"
	_ "github.com/signalfx/neo-agent/monitors/collectd/metadata"
	_ "github.com/signalfx/neo-agent/monitors/collectd/mongodb"
	_ "github.com/signalfx/neo-agent/monitors/collectd/mysql"
	_ "github.com/signalfx/neo-agent/monitors/collectd/nginx"
	_ "github.com/signalfx/neo-agent/monitors/collectd/rabbitmq"
	_ "github.com/signalfx/neo-agent/monitors/collectd/redis"
	_ "github.com/signalfx/neo-agent/monitors/collectd/zookeeper"
	_ "github.com/signalfx/neo-agent/monitors/kubernetes"
	_ "github.com/signalfx/neo-agent/observers/docker"
	_ "github.com/signalfx/neo-agent/observers/file"
	_ "github.com/signalfx/neo-agent/observers/kubelet"
	_ "github.com/signalfx/neo-agent/observers/kubernetes"
)
