package core

// Do an import of all of the built-in observers and monitors
// that apply to all platforms until we get a proper plugin system

import (
	// Import everything that isn't referenced anywhere else
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/appmesh"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/aspdotnet"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/cadvisor"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/consul"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/couchbase"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/elasticsearch"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/etcd"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/hadoop"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/haproxy"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/healthchecker"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/jenkins"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/kong"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/marathon"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/mongodb"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/openstack"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/python"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/rabbitmq"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/redis"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/spark"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/systemd"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/collectd/zookeeper"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/conviva"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/coredns"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/cpu"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/diskio"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/docker"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/dotnet"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/ecs"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/elasticsearch"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/etcd"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/expvar"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/filesystems"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/forwarder"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/gitlab"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/haproxy"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/heroku"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/internalmetrics"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/jaegergrpc"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/jmx"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/kubernetes"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/logstash/logstash"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/logstash/tcp"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/memory"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/metadata/hostmetadata"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/netio"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/postgresql"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/processlist"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/go"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/nginxvts"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/node"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/postgres"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/prometheus"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheus/redis"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/prometheusexporter"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/sql"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/statsd"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/subproc/signalfx/java"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/subproc/signalfx/python"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/mssqlserver"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/procstat"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/tail"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/telegraflogparser"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/telegrafsnmp"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/telegrafstatsd"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/winperfcounters"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/telegraf/monitors/winservices"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/traceforwarder"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/traefik"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/vmem"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/windowsiis"
	_ "github.com/signalfx/signalfx-agent/pkg/monitors/windowslegacy"
	_ "github.com/signalfx/signalfx-agent/pkg/observers/docker"
	_ "github.com/signalfx/signalfx-agent/pkg/observers/ecs"
	_ "github.com/signalfx/signalfx-agent/pkg/observers/host"
	_ "github.com/signalfx/signalfx-agent/pkg/observers/kubelet"
	_ "github.com/signalfx/signalfx-agent/pkg/observers/kubernetes"
)
