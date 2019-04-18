package core

// Do an import of all of the built-in observers and monitors
// that apply to all platforms until we get a proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/signalfx-agent/internal/monitors/aspdotnet"
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
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/systemd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/collectd/zookeeper"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/conviva"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/cpu"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/diskio"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/docker"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/dotnet"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/ecs"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/elasticsearch"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/expvar"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/filesystems"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/gitlab"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/internalmetrics"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/kubernetes"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/memory"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/metadata/hostmetadata"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/netio"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/postgresql"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/processlist"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/go"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/nginxvts"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/node"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/postgres"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/prometheus"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheus/redis"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/pyrunner/signalfx"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/sql"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/mssqlserver"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/procstat"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/tail"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/telegraflogparser"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/telegrafsnmp"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/telegrafstatsd"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winperfcounters"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/telegraf/monitors/winservices"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/traceforwarder"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/vmem"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/windowsiis"
	_ "github.com/signalfx/signalfx-agent/internal/monitors/windowslegacy"
	_ "github.com/signalfx/signalfx-agent/internal/observers/docker"
	_ "github.com/signalfx/signalfx-agent/internal/observers/ecs"
	_ "github.com/signalfx/signalfx-agent/internal/observers/file"
	_ "github.com/signalfx/signalfx-agent/internal/observers/host"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubelet"
	_ "github.com/signalfx/signalfx-agent/internal/observers/kubernetes"
)
