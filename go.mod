module github.com/signalfx/signalfx-agent

go 1.13

replace github.com/creasty/defaults => github.com/signalfx/defaults v1.2.2-0.20180531161417-70562fe60657

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

require (
	collectd.org v0.3.0 // indirect
	dmitri.shuralyov.com/app/changes v0.0.0-20180602232624-0a106ad413e3 // indirect
	dmitri.shuralyov.com/html/belt v0.0.0-20180602232347-f7d459c86be0 // indirect
	dmitri.shuralyov.com/service/change v0.0.0-20181023043359-a85b471d5412 // indirect
	dmitri.shuralyov.com/state v0.0.0-20180228185332-28bcc343414c // indirect
	github.com/Knetic/govaluate v2.3.0+incompatible
	github.com/Microsoft/go-winio v0.4.13
	github.com/ShowMax/go-fqdn v0.0.0-20160909083404-2501cdd51ef4
	github.com/StackExchange/wmi v0.0.0-20180725035823-b12b22c5341f
	github.com/araddon/gou v0.0.0-20190110011759-c797efecbb61 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e
	github.com/creasty/defaults v0.0.0-00010101000000-000000000000
	github.com/dancannon/gorethink v4.0.0+incompatible // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denisenkom/go-mssqldb v0.0.0-20190412130859-3b1d194e553a
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190316220345-38005cfc12fb
	github.com/docker/go-connections v0.4.0
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/go-playground/locales v0.11.2
	github.com/go-playground/universal-translator v0.16.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/go-stomp/stomp v2.0.2+incompatible // indirect
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/cadvisor v0.26.1
	github.com/gorilla/mux v1.7.3
	github.com/guregu/null v3.4.0+incompatible // indirect
	github.com/hashicorp/consul v1.6.2
	github.com/hashicorp/consul/api v1.3.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/hashicorp/nomad v0.8.7 // indirect
	github.com/hashicorp/vault v1.3.0
	github.com/hashicorp/vault-plugin-auth-gcp v0.5.2-0.20190930204802-acfd134850c2
	github.com/hashicorp/vault/api v1.0.5-0.20191108163347-bdd38fca2cff
	github.com/iancoleman/strcase v0.0.0-20171129010253-3de563c3dc08
	github.com/influxdata/platform v0.0.0-20190117200541-d500d3cf5589 // indirect
	github.com/influxdata/tail v1.0.0 // indirect
	github.com/influxdata/telegraf v0.10.2-0.20190319005412-5e88824c153e
	github.com/influxdata/toml v0.0.0-20180607005434-2a2e3012f7cf // indirect
	github.com/influxdata/wlog v0.0.0-20160411224016-7c63b0a71ef8 // indirect
	github.com/kardianos/service v1.0.0
	github.com/karrick/godirwalk v1.8.0 // indirect
	github.com/kr/pretty v0.1.0
	github.com/leodido/go-urn v1.1.0 // indirect
	github.com/lib/pq v1.2.0
	github.com/mailru/easyjson v0.7.0
	github.com/mattbaird/elastigo v0.0.0-20170123220020-2fe47fd29e4b // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-runewidth v0.0.6 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/michaelklishin/rabbit-hole v1.5.0 // indirect
	github.com/microcosm-cc/bluemonday v1.0.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0
	github.com/mitchellh/hashstructure v1.0.0
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/neelance/astrewrite v0.0.0-20160511093645-99348263ae86 // indirect
	github.com/neelance/sourcemap v0.0.0-20151028013722-8c68805598ab // indirect
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/openshift/client-go v3.9.0+incompatible
	github.com/ory-am/common v0.4.0 // indirect
	github.com/pkg/errors v0.8.2-0.20190227000051-27936f6d90f9
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.7.0
	github.com/samuel/go-zookeeper v0.0.0-20190810000440-0ceca61e4d75
	github.com/shirou/gopsutil v2.19.9+incompatible
	github.com/shurcooL/component v0.0.0-20170202220835-f88ec8f54cc4 // indirect
	github.com/shurcooL/events v0.0.0-20181021180414-410e4ca65f48 // indirect
	github.com/shurcooL/github_flavored_markdown v0.0.0-20181002035957-2122de532470 // indirect
	github.com/shurcooL/gofontwoff v0.0.0-20180329035133-29b52fc0a18d // indirect
	github.com/shurcooL/gopherjslib v0.0.0-20160914041154-feb6d3990c2c // indirect
	github.com/shurcooL/highlight_diff v0.0.0-20170515013008-09bb4053de1b // indirect
	github.com/shurcooL/highlight_go v0.0.0-20181028180052-98c3abbbae20 // indirect
	github.com/shurcooL/home v0.0.0-20181020052607-80b7ffcb30f9 // indirect
	github.com/shurcooL/htmlg v0.0.0-20170918183704-d01228ac9e50 // indirect
	github.com/shurcooL/httperror v0.0.0-20170206035902-86b7830d14cc // indirect
	github.com/shurcooL/httpgzip v0.0.0-20180522190206-b1c53ac65af9 // indirect
	github.com/shurcooL/issues v0.0.0-20181008053335-6292fdc1e191 // indirect
	github.com/shurcooL/issuesapp v0.0.0-20180602232740-048589ce2241 // indirect
	github.com/shurcooL/notifications v0.0.0-20181007000457-627ab5aea122 // indirect
	github.com/shurcooL/octicon v0.0.0-20181028054416-fa4f57f9efb2 // indirect
	github.com/shurcooL/reactions v0.0.0-20181006231557-f2e0b4ca5b82 // indirect
	github.com/shurcooL/sanitized_anchor_name v0.0.0-20170918181015-86672fcb3f95 // indirect
	github.com/shurcooL/users v0.0.0-20180125191416-49c67e49c537 // indirect
	github.com/shurcooL/webdavfs v0.0.0-20170829043945-18c3829fa133 // indirect
	github.com/signalfx/com_signalfx_metrics_protobuf v0.0.0-20190222193949-1fb69526e884
	github.com/signalfx/gateway v1.2.19-0.20191125135538-2c417b7ae0bd
	github.com/signalfx/golib/v3 v3.0.0
	github.com/signalfx/sapm-proto v0.0.0-00010101000000-000000000000
	github.com/signalfx/signalfx-go v1.6.9-0.20191121015807-da8b1dfaab43
	github.com/sirupsen/logrus v1.4.2
	github.com/soniah/gosnmp v0.0.0-20190220004421-68e8beac0db9 // indirect
	github.com/sourcegraph/annotate v0.0.0-20160123013949-f4cad6c6324d // indirect
	github.com/sourcegraph/syntaxhighlight v0.0.0-20170531221838-bd320f5d308e // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/tidwall/gjson v1.2.1 // indirect
	github.com/tidwall/match v1.0.1 // indirect
	github.com/uber/tchannel-go v1.16.0
	// github.com/ugorji/go v1.1.7
	github.com/ulule/deepcopier v0.0.0-20171107155558-ca99b135e50f
	github.com/vjeantet/grok v1.0.0 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	// go.etcd.io/etcd v0.0.0-20190321122103-41f7142ff986
	go.etcd.io/etcd v3.3.18+incompatible
	golang.org/x/crypto v0.0.0-20191107222254-f4817d981bb6 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20191105231009-c1f44814a5cd
	gopkg.in/fatih/set.v0 v0.1.0
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v9 v9.28.0
	gopkg.in/gorethink/gorethink.v4 v4.1.0 // indirect
	gopkg.in/ory-am/dockertest.v2 v2.2.3 // indirect
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.0.0-20190813020757-36bff7324fb7
	k8s.io/apimachinery v0.0.0-20190809020650-423f5d784010
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubernetes v1.12.0
	sourcegraph.com/sourcegraph/go-diff v0.5.0 // indirect
)

replace github.com/signalfx/sapm-proto => ../sapm-proto

// replace github.com/ugorji/go/codec => github.com/ugorji/go/codec v1.1.7

replace k8s.io/api => k8s.io/api v0.0.0-20181110191121-a33c8200050f

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20180621070125-103fd098999d

replace k8s.io/client-go => k8s.io/client-go v8.0.0+incompatible

replace github.com/jaegertracing/jaeger => github.com/jaegertracing/jaeger v1.7.0
