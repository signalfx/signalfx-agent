// +build !windows

package core

// Do an import of all of the built-in observers and monitors that are
// not appropriate for windows until we get a proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/activemq"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/apache"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/cassandra"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/chrony"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/consul"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/couchbase"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/cpu"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/cpufreq"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/custom"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/df"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/disk"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/docker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/elasticsearch"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/etcd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/genericjmx"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/haproxy"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/healthchecker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/jenkins"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/kafka"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/kong"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/load"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/marathon"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/memcached"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/memory"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/metadata"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/mongodb"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/mysql"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/netinterface"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/nginx"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/openstack"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/postgresql"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/processes"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/protocols"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/rabbitmq"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/redis"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/spark"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/statsd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/uptime"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/vmem"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/zookeeper"
)
