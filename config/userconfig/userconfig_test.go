package userconfig

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestUserConfig_LoadYAML(t *testing.T) {
	var one = int(1)
	var two = int(2)
	var three = int(3)
	var four = int(4)
	var five = int(5)
	var tru = true
	type fields struct {
		Collectd   *Collectd
		Filter     *Filter
		IngestURL  string
		Kubernetes *Kubernetes
		Mesosphere *Mesosphere
		Proxy      *Proxy
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"",
			fields{
				IngestURL: "http://www.ingesturl.com",
				Proxy: &Proxy{
					HTTP:  "http://www.notarealproxy.com",
					HTTPS: "https://www.notarealproxy.com",
					Skip:  "http://www.skipthis.com",
				},
				Filter: &Filter{
					[]string{ // dockerContainerNames
						"^somethingsnarky$",
					},
					[]string{ // images
						"^elasticsearch:*",
						"^mysql:*",
					},
					[]string{ // KubernetesContainerNames
						"^redis:*",
					},
					[]string{ // KubernetesPodNames
						"^podNameOne$",
					},
					[]string{ // KubernetesNamespaces
						"^default$",
					},
					[]*Label{ // Labels
						&Label{
							Key:   "^environment$",
							Value: "^dev$",
						},
						&Label{
							Key:   "^environment$",
							Value: "^test$",
						},
					},
				},
				Collectd: &Collectd{
					Interval:             &one,
					Timeout:              &two,
					ReadThreads:          &three,
					WriteQueueLimitHigh:  &four,
					WriteQueueLimitLow:   &five,
					CollectInternalStats: &tru,
				},
				Kubernetes: &Kubernetes{
					Role:        "worker",
					Cluster:     "kubernetes-cluster",
					CAdvisorURL: "http://localhost:4493",
					CAdvisorMetricFilter: []string{
						"container_cpu_utilization",
						"container_cpu_utilization_per_core",
					},
					CAdvisorDataSendRate: 25,
					KubernetesAPI: &struct {
						AuthType string `yaml:"authType,omitempty"`
						TLS *TLS `yaml:"tls,omitempty"`
					}{
						TLS: &TLS{
							SkipVerify: false,
							ClientCert: "/path/to/cert",
							ClientKey:  "/path/to/key",
							CACert:     "/path/to/ca",
						},
					},
					KubeletAPI: &struct {
						TLS *TLS `yaml:"tls,omitempty"`
					}{
						TLS: &TLS{
							SkipVerify: true,
							ClientCert: "/path/to/cert",
							ClientKey:  "/path/to/key",
							CACert:     "/path/to/ca",
						},
					},
				},
				Mesosphere: nil,
			},
			args{"testdata/userconfig/userconfig-1.yaml"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserConfig{
				Collectd:   tt.fields.Collectd,
				Filter:     tt.fields.Filter,
				IngestURL:  tt.fields.IngestURL,
				Kubernetes: tt.fields.Kubernetes,
				Mesosphere: tt.fields.Mesosphere,
				Proxy:      tt.fields.Proxy,
			}
			var userConfig = &UserConfig{}
			if err := userConfig.LoadYAML(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("UserConfig.LoadYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(userConfig, u) {
				pretty.Ldiff(t, userConfig, u)
				t.Error("UserConfig.LoadYAML() Differences detected")
			}
		})
	}
}
