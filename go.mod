module github.com/signalfx/signalfx-agent

go 1.14

replace (
	code.cloudfoundry.org/go-loggregator => github.com/signalfx/go-loggregator v1.0.1-0.20200205155641-5ba5ca92118d
	github.com/dancannon/gorethink => gopkg.in/gorethink/gorethink.v4 v4.0.0
	github.com/influxdata/telegraf => github.com/signalfx/telegraf v0.10.2-0.20211029142026-90d18852ba43
	github.com/signalfx/signalfx-agent/pkg/apm => ./pkg/apm
	github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.5-0.20210205191134-5ec6847320e5 // required to drop google.golang.org/grpc/examples/helloworld/helloworld test dep
	google.golang.org/grpc => google.golang.org/grpc v1.29.1 // required to provide google.golang.org/grpc/naming to satisfy go.etcd.io/etcd test dep
)

// security updates
replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.12
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.0-rc95
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1
)

require (
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	collectd.org v0.5.0 // indirect
	github.com/Microsoft/go-winio v0.4.16
	github.com/SAP/go-hdb v0.105.2
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/Showmax/go-fqdn v1.0.0
	github.com/StackExchange/wmi v1.2.1
	github.com/antonmedv/expr v1.8.9
	github.com/aws/aws-sdk-go v1.38.3
	github.com/beevik/ntp v0.3.0
	github.com/cloudfoundry-incubator/uaago v0.0.0-20190307164349-8136b7bbe76e
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/creasty/defaults v1.5.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.11.0
	github.com/docker/docker v20.10.5+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/go-errors/errors v1.0.1
	github.com/go-playground/locales v0.12.1
	github.com/go-playground/universal-translator v0.16.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-test/deep v1.0.7
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.2
	github.com/google/cadvisor v0.26.1
	github.com/gorilla/mux v1.8.0
	github.com/guregu/null v4.0.0+incompatible // indirect
	github.com/hashicorp/consul/api v1.8.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/vault v1.7.6 // required for newer google.golang.org/api compatibility
	github.com/hashicorp/vault-plugin-auth-gcp v0.9.1
	github.com/hashicorp/vault/api v1.1.1
	github.com/iancoleman/strcase v0.0.0-20171129010253-3de563c3dc08
	github.com/influxdata/tail v1.0.0 // indirect
	github.com/influxdata/telegraf v0.0.0-00010101000000-000000000000
	github.com/influxdata/toml v0.0.0-20180607005434-2a2e3012f7cf // indirect
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8 // indirect
	github.com/jackc/pgx/v4 v4.6.0
	github.com/jaegertracing/jaeger v1.26.0
	github.com/kardianos/service v1.0.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kr/pretty v0.3.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/lib/pq v1.10.3
	github.com/mailru/easyjson v0.7.7
	github.com/mattn/go-xmlrpc v0.0.3
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/mongodb/go-client-mongodb-atlas v0.2.0
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.4
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20201020071134-e303d21b3e32 // to make compatible w/ k8s.io/client-go v0.19.4
	github.com/opentracing/opentracing-go v1.2.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.6.0
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e
	github.com/shirou/gopsutil v3.21.5+incompatible
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.2
	github.com/signalfx/defaults v1.2.2-0.20180531161417-70562fe60657
	github.com/signalfx/gateway v1.2.19-0.20191125135538-2c417b7ae0bd
	github.com/signalfx/golib/v3 v3.3.43
	github.com/signalfx/ingest-protocols v0.1.3
	github.com/signalfx/signalfx-agent/pkg/apm v0.0.0-00010101000000-000000000000
	github.com/signalfx/signalfx-go v1.6.38-0.20200518153434-ceee8d2570d5
	github.com/signalfx/signalfx-go-tracing v1.2.0
	github.com/sirupsen/logrus v1.8.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/snowflakedb/gosnowflake v1.4.3
	github.com/soniah/gosnmp v0.0.0-20190220004421-68e8beac0db9 // indirect; required; first version with go modules
	github.com/stretchr/testify v1.7.0
	github.com/tidwall/gjson v1.9.3 // indirect
	github.com/tinylib/msgp v1.1.5 // indirect
	github.com/ulule/deepcopier v0.0.0-20171107155558-ca99b135e50f
	github.com/vjeantet/grok v1.0.0 // indirect
	github.com/vmware/govmomi v0.23.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd/client/v2 v2.305.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210816074244-15123e1e1f71
	golang.org/x/tools v0.1.2
	google.golang.org/grpc v1.40.0
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/go-playground/validator.v9 v9.29.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/kubelet v0.20.5
)
