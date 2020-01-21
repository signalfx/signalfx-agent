module github.com/signalfx/signalfx-agent

go 1.13

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

require (
	collectd.org v0.3.0 // indirect
	github.com/Azure/azure-sdk-for-go v26.4.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.4.2 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Knetic/govaluate v2.3.0+incompatible
	github.com/Microsoft/go-winio v0.4.11
	github.com/NYTimes/gziphandler v1.1.1 // indirect
	github.com/SAP/go-hdb v0.14.1 // indirect
	github.com/ShowMax/go-fqdn v0.0.0-20160909083404-2501cdd51ef4
	github.com/StackExchange/wmi v0.0.0-20180725035823-b12b22c5341f
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20190315122603-6f9e54af456e // indirect
	github.com/araddon/gou v0.0.0-20190110011759-c797efecbb61 // indirect
	github.com/aws/aws-sdk-go v1.18.4 // indirect
	github.com/boombuler/barcode v1.0.0 // indirect
	github.com/briankassouf/jose v0.9.1 // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible // indirect
	github.com/centrify/cloud-golang-sdk v0.0.0-20190214225812-119110094d0f // indirect
	github.com/chrismalek/oktasdk-go v0.0.0-20181212195951-3430665dfaa0 // indirect
	github.com/containerd/continuity v0.0.0-20181203112020-004b46473808 // indirect
	github.com/coreos/go-oidc v2.0.0+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/creasty/defaults v1.3.0 // indirect
	github.com/dancannon/gorethink v4.0.0+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.0.0-20190121005146-b04fd42d9952
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190316220345-38005cfc12fb
	github.com/docker/go-connections v0.4.0
	github.com/duosecurity/duo_api_golang v0.0.0-20190308151101-6c680f768e74 // indirect
	github.com/fullsailor/pkcs7 v0.0.0-20180613152042-8306686428a5 // indirect
	github.com/gammazero/deque v0.0.0-20190130191400-2afb3858e9c7 // indirect
	github.com/gammazero/workerpool v0.0.0-20181230203049-86a96b5d5d92 // indirect
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-ldap/ldap v3.0.2+incompatible // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/go-playground/locales v0.11.2
	github.com/go-playground/universal-translator v0.16.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/go-stomp/stomp v2.0.2+incompatible // indirect
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gocql/gocql v0.0.0-20190301043612-f6df8288f9b4 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/google/cadvisor v0.26.1
	github.com/googleapis/gnostic v0.1.0 // indirect
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75 // indirect
	github.com/gorilla/mux v1.6.1
	github.com/guregu/null v3.4.0+incompatible // indirect
	github.com/hashicorp/consul v1.4.0
	github.com/hashicorp/go-gcp-common v0.0.0-20180425173946-763e39302965 // indirect
	github.com/hashicorp/go-hclog v0.8.0 // indirect
	github.com/hashicorp/go-memdb v0.0.0-20190306140544-eea0b16292ad // indirect
	github.com/hashicorp/go-plugin v0.0.0-20190220160451-3f118e8ee104 // indirect
	github.com/hashicorp/go-retryablehttp v0.5.2 // indirect
	github.com/hashicorp/go-uuid v1.0.1 // indirect
	github.com/hashicorp/go-version v1.1.0 // indirect
	github.com/hashicorp/golang-lru v0.5.1
	github.com/hashicorp/memberlist v0.1.3 // indirect
	github.com/hashicorp/nomad v0.8.7 // indirect
	github.com/hashicorp/vault v1.1.1-0.20190321125746-66ef59957aaf
	github.com/hashicorp/vault-plugin-auth-alicloud v0.0.0-20190311155555-98628998247d // indirect
	github.com/hashicorp/vault-plugin-auth-azure v0.0.0-20190201222632-0af1d040b5b3 // indirect
	github.com/hashicorp/vault-plugin-auth-centrify v0.0.0-20180816201131-66b0a34a58bf // indirect
	github.com/hashicorp/vault-plugin-auth-gcp v0.0.0-20190320214413-e8308b5e41c9
	github.com/hashicorp/vault-plugin-auth-jwt v0.0.0-20190314211503-86b44673ce1e // indirect
	github.com/hashicorp/vault-plugin-auth-kubernetes v0.0.0-20190201222209-db96aa4ab438 // indirect
	github.com/hashicorp/vault-plugin-secrets-ad v0.0.0-20190131222416-4796d9980125 // indirect
	github.com/hashicorp/vault-plugin-secrets-alicloud v0.0.0-20190131211812-b0abe36195cb // indirect
	github.com/hashicorp/vault-plugin-secrets-azure v0.0.0-20181207232500-0087bdef705a // indirect
	github.com/hashicorp/vault-plugin-secrets-gcp v0.0.0-20190311200649-621231cb86fe // indirect
	github.com/hashicorp/vault-plugin-secrets-gcpkms v0.0.0-20190116164938-d6b25b0b4a39 // indirect
	github.com/hashicorp/vault-plugin-secrets-kv v0.0.0-20190315192709-dccffee64925 // indirect
	github.com/iancoleman/strcase v0.0.0-20171129010253-3de563c3dc08
	github.com/influxdata/influxdb v1.7.4 // indirect
	github.com/influxdata/platform v0.0.0-20190117200541-d500d3cf5589 // indirect
	github.com/influxdata/tail v1.0.0 // indirect
	github.com/influxdata/telegraf v0.10.2-0.20200121190823-6dad859d74c2
	github.com/influxdata/toml v0.0.0-20180607005434-2a2e3012f7cf // indirect
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8 // indirect
	github.com/jaegertracing/jaeger v1.15.1
	github.com/jeffchao/backoff v0.0.0-20140404060208-9d7fd7aa17f2 // indirect
	github.com/jefferai/jsonx v1.0.0 // indirect
	github.com/kardianos/service v1.0.0
	github.com/karrick/godirwalk v1.8.0 // indirect
	github.com/keybase/go-crypto v0.0.0-20190312101036-b475f2ecc1fe // indirect
	github.com/kr/pretty v0.1.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lib/pq v1.0.0
	github.com/mailru/easyjson v0.7.0
	github.com/mattbaird/elastigo v0.0.0-20170123220020-2fe47fd29e4b // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.6 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/michaelklishin/rabbit-hole v1.5.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v0.0.0-20170609045927-2bca23e0e452
	github.com/mitchellh/pointerstructure v0.0.0-20170205204203-f2329fcfa9e2 // indirect
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v0.0.0-20191216194936-57f413491e9e
	github.com/opentracing/opentracing-go v1.1.0
	github.com/ory-am/common v0.4.0 // indirect
	github.com/ory/dockertest v3.3.4+incompatible // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pkg/errors v0.8.2-0.20190227000051-27936f6d90f9
	github.com/pquerna/cachecontrol v0.0.0-20180517163645-1555304b9b35 // indirect
	github.com/pquerna/otp v1.1.0 // indirect
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.2.1-0.20190321124555-1ab4d74fc899
	github.com/prometheus/procfs v0.0.9-0.20191209220459-fa4d6ce8c078
	github.com/samuel/go-zookeeper v0.0.0-20190810000440-0ceca61e4d75
	github.com/shirou/gopsutil v2.19.12+incompatible
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.0-20190222193949-1fb69526e884
	github.com/signalfx/defaults v1.2.2-0.20180531161417-70562fe60657
	github.com/signalfx/gateway v1.2.19-0.20191125135538-2c417b7ae0bd
	github.com/signalfx/golib/v3 v3.1.0
	github.com/signalfx/signalfx-go v1.6.9-0.20191121015807-da8b1dfaab43
	github.com/sirupsen/logrus v1.4.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/soniah/gosnmp v1.22.0 // indirect
	github.com/streadway/amqp v0.0.0-20190312223743-14f78b41ce6d // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.2.1 // indirect
	github.com/tidwall/match v1.0.1 // indirect
	github.com/tidwall/pretty v0.0.0-20180105212114-65a9db5fad51 // indirect
	github.com/ugorji/go/codec v0.0.0-20190320090025-2dc34c0b8780 // indirect
	github.com/ulule/deepcopier v0.0.0-20171107155558-ca99b135e50f
	github.com/vjeantet/grok v1.0.0 // indirect
	github.com/vmware/govmomi v0.21.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	go.etcd.io/etcd v0.0.0-20190321122103-41f7142ff986
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd
	google.golang.org/grpc v1.20.1
	gopkg.in/asn1-ber.v1 v1.0.0-20181015200546-f715ec2f112d // indirect
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/gorethink/gorethink.v4 v4.1.0 // indirect
	gopkg.in/ini.v1 v1.42.0 // indirect
	gopkg.in/ory-am/dockertest.v2 v2.2.3 // indirect
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	k8s.io/kubernetes v1.12.0
	layeh.com/radius v0.0.0-20190118135028-0f678f039617 // indirect
)
