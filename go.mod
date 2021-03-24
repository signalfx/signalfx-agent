module github.com/signalfx/signalfx-agent

go 1.14

replace (
	code.cloudfoundry.org/go-loggregator => github.com/signalfx/go-loggregator v1.0.1-0.20200205155641-5ba5ca92118d
	github.com/dancannon/gorethink => gopkg.in/gorethink/gorethink.v4 v4.0.0
	github.com/influxdata/telegraf => github.com/signalfx/telegraf v0.10.2-0.20210126144230-e303a54ab07d
	github.com/signalfx/signalfx-agent/pkg/apm => ./pkg/apm
	github.com/soheilhy/cmux => ./thirdparty/cmux // required to drop google.golang.org/grpc/examples/helloworld/helloworld test dep
	google.golang.org/grpc => google.golang.org/grpc v1.29.1 // required to provide google.golang.org/grpc/naming to satisfy go.etcd.io/etcd test dep
)

require (
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	collectd.org v0.5.0 // indirect
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/Showmax/go-fqdn v1.0.0
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d
	github.com/antonmedv/expr v1.8.9
	github.com/aws/aws-sdk-go v1.38.0
	github.com/beevik/ntp v0.3.0
	github.com/cloudfoundry-incubator/uaago v0.0.0-20190307164349-8136b7bbe76e
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/creasty/defaults v1.5.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.0.0-20200428022330-06a60b6afbbc
	github.com/docker/docker v17.12.0-ce-rc1.0.20200706150819-a40b877fbb9e+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/go-errors/errors v1.0.1
	github.com/go-playground/locales v0.11.2
	github.com/go-playground/universal-translator v0.16.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-test/deep v1.0.7
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.1
	github.com/google/cadvisor v0.26.1
	github.com/gorilla/mux v1.8.0
	github.com/guregu/null v4.0.0+incompatible // indirect
	github.com/hashicorp/consul/api v1.7.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/vault v1.6.0 // required for newer google.golang.org/api compatibility
	github.com/hashicorp/vault-plugin-auth-gcp v0.8.0
	github.com/hashicorp/vault/api v1.0.5-0.20201001211907-38d91b749c77
	github.com/iancoleman/strcase v0.0.0-20171129010253-3de563c3dc08
	github.com/influxdata/tail v1.0.0 // indirect
	github.com/influxdata/telegraf v0.0.0-00010101000000-000000000000
	github.com/influxdata/toml v0.0.0-20180607005434-2a2e3012f7cf // indirect
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8 // indirect
	github.com/jackc/pgx/v4 v4.6.0
	github.com/jaegertracing/jaeger v1.21.0
	github.com/kardianos/service v1.0.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kr/pretty v0.2.1
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/lib/pq v1.8.0
	github.com/mailru/easyjson v0.7.1
	github.com/mattn/go-xmlrpc v0.0.3
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mongodb/go-client-mongodb-atlas v0.2.0
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20201020071134-e303d21b3e32 // to make compatible w/ k8s.io/client-go v0.19.4
	github.com/opentracing/opentracing-go v1.2.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.15.0
	github.com/prometheus/procfs v0.6.0
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e
	github.com/shirou/gopsutil v3.20.10+incompatible
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.2
	github.com/signalfx/defaults v1.2.2-0.20180531161417-70562fe60657
	github.com/signalfx/gateway v1.2.19-0.20191125135538-2c417b7ae0bd
	github.com/signalfx/golib/v3 v3.3.16
	github.com/signalfx/ingest-protocols v0.0.16
	github.com/signalfx/signalfx-agent/pkg/apm v0.0.0-00010101000000-000000000000
	github.com/signalfx/signalfx-go v1.6.38-0.20200518153434-ceee8d2570d5
	github.com/signalfx/signalfx-go-tracing v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/soniah/gosnmp v0.0.0-20190220004421-68e8beac0db9 // indirect; required; first version with go modules
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.4 // indirect
	github.com/tinylib/msgp v1.1.5 // indirect
	github.com/ulule/deepcopier v0.0.0-20171107155558-ca99b135e50f
	github.com/vjeantet/grok v1.0.0 // indirect
	github.com/vmware/govmomi v0.23.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd v0.5.0-alpha.5.0.20200425165423-262c93980547
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	golang.org/x/tools v0.0.0-20201022035929-9cf592e881e9
	google.golang.org/grpc v1.29.1
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/yaml.v2 v2.3.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kubernetes v1.12.0
)
