module github.com/signalfx/signalfx-agent

go 1.17

replace (
	code.cloudfoundry.org/go-loggregator => github.com/signalfx/go-loggregator v1.0.1-0.20200205155641-5ba5ca92118d
	github.com/dancannon/gorethink => gopkg.in/gorethink/gorethink.v4 v4.0.0
	github.com/influxdata/telegraf => github.com/signalfx/telegraf v0.10.2-0.20211029142026-90d18852ba43
	github.com/signalfx/signalfx-agent/pkg/apm => ./pkg/apm
	github.com/soheilhy/cmux => github.com/soheilhy/cmux v0.1.5-0.20210205191134-5ec6847320e5 // required to drop google.golang.org/grpc/examples/helloworld/helloworld test dep
)

// security updates
replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.13
	github.com/go-kit/kit => github.com/go-kit/kit v0.12.0
	github.com/nats-io/jwt/v2 => github.com/nats-io/jwt/v2 v2.2.0
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3
	github.com/spf13/viper => github.com/spf13/viper v1.11.0 // required to drop dependency on deprecated github.com/coreos/etcd and github.com/coreos/go-etcd
	go.mongodb.org/mongo-driver => go.mongodb.org/mongo-driver v1.5.1
)

require (
	code.cloudfoundry.org/go-loggregator v7.4.0+incompatible
	github.com/Microsoft/go-winio v0.5.1
	github.com/SAP/go-hdb v0.105.2
	github.com/Sectorbob/mlab-ns2 v0.0.0-20171030222938-d3aa0c295a8a
	github.com/Showmax/go-fqdn v1.0.0
	github.com/StackExchange/wmi v1.2.1
	github.com/antonmedv/expr v1.8.9
	github.com/aws/aws-sdk-go v1.40.45
	github.com/beevik/ntp v0.3.0
	github.com/cloudfoundry-incubator/uaago v0.0.0-20190307164349-8136b7bbe76e
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.12.0
	github.com/docker/docker v20.10.10+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/go-errors/errors v1.4.1
	github.com/go-playground/locales v0.12.1
	github.com/go-playground/universal-translator v0.16.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/go-test/deep v1.0.8
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.2
	github.com/google/cadvisor v0.26.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.12.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/vault v1.10.2 // required for newer google.golang.org/api compatibility
	github.com/hashicorp/vault-plugin-auth-gcp v0.12.1
	github.com/hashicorp/vault/api v1.5.0
	github.com/iancoleman/strcase v0.0.0-20171129010253-3de563c3dc08
	github.com/influxdata/telegraf v0.0.0-00010101000000-000000000000
	github.com/jackc/pgx/v4 v4.6.0
	github.com/jaegertracing/jaeger v1.26.0
	github.com/kardianos/service v1.0.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kr/pretty v0.3.0
	github.com/lib/pq v1.10.3
	github.com/mailru/easyjson v0.7.7
	github.com/mattn/go-xmlrpc v0.0.3
	github.com/mauricelam/genny v0.0.0-20190320071652-0800202903e5
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/mongodb/go-client-mongodb-atlas v0.2.0
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20201020071134-e303d21b3e32 // to make compatible w/ k8s.io/client-go v0.19.4
	github.com/opentracing/opentracing-go v1.2.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.7.3
	github.com/samuel/go-zookeeper v0.0.0-20200724154423-2164a8ac840e
	github.com/shirou/gopsutil v3.21.5+incompatible
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.2
	github.com/signalfx/defaults v1.2.2-0.20180531161417-70562fe60657
	github.com/signalfx/gateway v1.2.23
	github.com/signalfx/golib/v3 v3.3.44
	github.com/signalfx/ingest-protocols v0.1.3
	github.com/signalfx/signalfx-agent/pkg/apm v0.0.0-00010101000000-000000000000
	github.com/signalfx/signalfx-go v1.6.38-0.20200518153434-ceee8d2570d5
	github.com/signalfx/signalfx-go-tracing v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/smartystreets/goconvey v1.6.4
	github.com/snowflakedb/gosnowflake v1.6.3
	github.com/stretchr/testify v1.7.1
	github.com/ulule/deepcopier v0.0.0-20171107155558-ca99b135e50f
	github.com/vmware/govmomi v0.23.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0
	go.etcd.io/etcd/client/v2 v2.305.2
	golang.org/x/net v0.0.0-20220412020605-290c469a71a5
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad
	golang.org/x/tools v0.1.5
	google.golang.org/grpc v1.45.0
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/go-playground/validator.v9 v9.29.1
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.24.0
	k8s.io/apimachinery v0.24.0
	k8s.io/client-go v0.24.0
	k8s.io/kubelet v0.20.5
)

require (
	code.cloudfoundry.org/go-diodes v0.0.0-20180905200951-72629b5276e3 // indirect
	code.cloudfoundry.org/rfc5424 v0.0.0-20180905210152-236a6d29298a // indirect
	collectd.org v0.5.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/apache/arrow/go/arrow v0.0.0-20210818145353-234c94e4ce64 // indirect
	github.com/apache/thrift v0.14.2 // indirect
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.16.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.3.2 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.5.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.12.0 // indirect
	github.com/aws/smithy-go v1.11.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/containerd v1.5.7 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/creasty/defaults v1.5.1 // indirect
	github.com/docker/distribution v2.8.0+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.3.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/guregu/null v4.0.0+incompatible // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-gcp-common v0.7.0 // indirect
	github.com/hashicorp/go-hclog v1.2.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-kms-wrapping/entropy v0.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/go-version v1.4.0 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-3 // indirect
	github.com/hashicorp/serf v0.9.7 // indirect
	github.com/hashicorp/vault/sdk v0.4.2-0.20220422204215-a7cef09c9429 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/influxdata/tail v1.0.0 // indirect
	github.com/influxdata/toml v0.0.0-20180607005434-2a2e3012f7cf // indirect
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.5.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200307190119-3430c5407db8 // indirect
	github.com/jackc/pgtype v1.3.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/karrick/godirwalk v1.10.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mwielbut/pointy v1.1.0 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/philhofer/fwd v1.1.1 // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.6.2 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shirou/gopsutil/v3 v3.21.9 // indirect
	github.com/signalfx/gohistogram v0.0.0-20160107210732-1ccfd2ff5083 // indirect
	github.com/signalfx/golib v2.4.1+incompatible // indirect
	github.com/signalfx/sapm-proto v0.7.2 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/soniah/gosnmp v0.0.0-20190220004421-68e8beac0db9 // indirect; required; first version with go modules
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/tidwall/gjson v1.9.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tinylib/msgp v1.1.5 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/vjeantet/grok v1.0.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.4 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.4 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20220411220226-7b82a4e95df4 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/oauth2 v0.0.0-20220411215720-9780585627b5 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20220210224613-90d013bbcef8 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	google.golang.org/api v0.74.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220413183235-5e96e2839df9 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/klog/v2 v2.60.1 // indirect
	k8s.io/kube-openapi v0.0.0-20220328201542-3ee0da9b0b42 // indirect
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

require (
	cloud.google.com/go/compute v1.5.0 // indirect
	cloud.google.com/go/kms v1.4.0 // indirect
	cloud.google.com/go/monitoring v1.5.0 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.11 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/gabriel-vasile/mimetype v1.3.1 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/golang-sql/sqlexp v0.0.0-20170517235910-f1bb20e5a188 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/hashicorp/go-secure-stdlib/awsutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.4 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.8 // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
)
