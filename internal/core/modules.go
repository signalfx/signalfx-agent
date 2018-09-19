package core

// Do an import of all of the built-in observers and monitors
// that apply to all platforms until we get a proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/signalfx-agent/internal/monitors/cadvisor"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/consul"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/couchbase"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/elasticsearch"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/etcd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/hadoop"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/haproxy"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/healthchecker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/jenkins"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/kong"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/marathon"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/mongodb"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/openstack"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/python"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/rabbitmq"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/redis"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/spark"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/zookeeper"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/conviva"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/docker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/internalmetrics"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/metadata/hostmetadata"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/processlist"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/mssqlserver"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/procstat"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/tail"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/telegrafstatsd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/traceforwarder"
	_ "github.com/signalfx/signalfx-agent/internal/observers/docker"
	_ "github.com/signalfx/signalfx-agent/internal/observers/file"
	_ "github.com/signalfx/signalfx-agent/internal/observers/host"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubelet"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubernetes"
)
