package userconfig

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestKubernetes_IsValid(t *testing.T) {
	type fields struct {
		TLS                  *TLS
		Role                 string
		Cluster              string
		CAdvisorURL          string
		CAdvisorMetricFilter []string
		CAdvisorDataSendRate int
	}
	tests := []struct {
		name         string
		testDataFile string
		want         bool
		wantErr      bool
	}{
		{
			"Kubernetes.IsValid() valid configuration",
			"testdata/kubernetes/kubernetes-valid.yaml",
			true,
			false,
		},
		{
			"Kubernetes.IsValid() invalid configuration",
			"testdata/kubernetes/kubernetes-invalid.yaml",
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var k = &Kubernetes{}
			var err error
			if err = k.LoadYAML(tt.testDataFile); err != nil {
				t.Error("Kubernetes.LoadYAML() Unable to load test data")
			}
			got, err := k.IsValid()
			if (err != nil) != tt.wantErr {
				t.Errorf("Kubernetes.IsValid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Kubernetes.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubernetes_Parse(t *testing.T) {
	type fields struct {
		TLS                  *TLS
		Role                 string
		Cluster              string
		CAdvisorURL          string
		CAdvisorMetricFilter []string
		CAdvisorDataSendRate int
	}
	type args struct {
		testData   string
		kubernetes map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		parsed  map[string]interface{}
		args    args
		wantErr bool
	}{
		{
			"Kubernetes.Parse() valid worker",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/cert",
					ClientKey:  "/path/to/key",
					CACert:     "/path/to/ca",
				},
				Role:        "worker",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:4493",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 29,
			},
			map[string]interface{}{
				"tls": map[string]interface{}{
					"skipVerify": true,
					"clientCert": "/path/to/cert",
					"clientKey":  "/path/to/key",
					"caCert":     "/path/to/ca",
				},
			},
			args{
				testData:   "testdata/kubernetes/kubernetes-valid-worker.yaml",
				kubernetes: map[string]interface{}{},
			},
			false,
		},
		{
			"Kubernetes.Parse() valid master",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/certAlt",
					ClientKey:  "/path/to/keyAlt",
					CACert:     "/path/to/caAlt",
				},
				Role:        "worker",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:8080",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 30,
			},
			map[string]interface{}{
				"tls": map[string]interface{}{
					"skipVerify": true,
					"clientCert": "/path/to/certAlt",
					"clientKey":  "/path/to/keyAlt",
					"caCert":     "/path/to/caAlt",
				},
			},
			args{
				testData:   "testdata/kubernetes/kubernetes-valid-alt-worker.yaml",
				kubernetes: map[string]interface{}{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kubernetes{
				TLS:                  tt.fields.TLS,
				Role:                 tt.fields.Role,
				Cluster:              tt.fields.Cluster,
				CAdvisorURL:          tt.fields.CAdvisorURL,
				CAdvisorMetricFilter: tt.fields.CAdvisorMetricFilter,
				CAdvisorDataSendRate: tt.fields.CAdvisorDataSendRate,
			}
			var kubernetes = &Kubernetes{}
			var err error
			if err = kubernetes.LoadYAML(tt.args.testData); err == nil {
				if !reflect.DeepEqual(kubernetes, k) {
					pretty.Ldiff(t, kubernetes, k)
					t.Error("Kubernetes.LoadYAML() Differences detected")
				}
			}
			if err := kubernetes.Parse(tt.args.kubernetes); (err != nil) != tt.wantErr {
				t.Errorf("Kubernetes.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.parsed, tt.args.kubernetes) {
				pretty.Ldiff(t, tt.parsed, tt.args.kubernetes)
				t.Error("Kubernetes.Parse() Differences detected")
			}
		})
	}
}

func TestKubernetes_ParseDimensions(t *testing.T) {
	type fields struct {
		TLS                  *TLS
		Role                 string
		Cluster              string
		CAdvisorURL          string
		CAdvisorMetricFilter []string
		CAdvisorDataSendRate int
	}
	type args struct {
		testData string
		dims     map[string]string
	}
	tests := []struct {
		name         string
		fields       fields
		expectedDims map[string]string
		args         args
		wantErr      bool
	}{
		{
			"Kubernetes.ParseDimensions() valid worker",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/cert",
					ClientKey:  "/path/to/key",
					CACert:     "/path/to/ca",
				},
				Role:        "worker",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:4493",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 29,
			},
			map[string]string{
				"kubernetes_cluster": "kubernetes-cluster",
				"kubernetes_role":    "worker",
			},
			args{
				testData: "testdata/kubernetes/kubernetes-valid-worker.yaml",
				dims:     map[string]string{},
			},
			false,
		},
		{
			"Kubernetes.ParseDimensions() valid master",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/certMaster",
					ClientKey:  "/path/to/keyMaster",
					CACert:     "/path/to/caMaster",
				},
				Role:        "master",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:8080",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 30,
			},
			map[string]string{
				"kubernetes_cluster": "kubernetes-cluster",
				"kubernetes_role":    "master",
			},
			args{
				testData: "testdata/kubernetes/kubernetes-valid-master.yaml",
				dims:     map[string]string{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kubernetes{
				TLS:                  tt.fields.TLS,
				Role:                 tt.fields.Role,
				Cluster:              tt.fields.Cluster,
				CAdvisorURL:          tt.fields.CAdvisorURL,
				CAdvisorMetricFilter: tt.fields.CAdvisorMetricFilter,
				CAdvisorDataSendRate: tt.fields.CAdvisorDataSendRate,
			}

			var kubernetes = &Kubernetes{}
			var err error
			if err = kubernetes.LoadYAML(tt.args.testData); err == nil {
				if !reflect.DeepEqual(kubernetes, k) {
					pretty.Ldiff(t, kubernetes, k)
					t.Error("Kubernetes.ParseDimensions() Differences detected")
				}
			}

			if err := kubernetes.ParseDimensions(tt.args.dims); (err != nil) != tt.wantErr {
				t.Errorf("Kubernetes.ParseDimensions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.expectedDims, tt.args.dims) {
				pretty.Ldiff(t, tt.expectedDims, tt.args.dims)
				t.Error("Kubernetes.ParseDimensions() Differences detected")
			}
		})
	}
}

func TestKubernetes_ParseCAdvisor(t *testing.T) {
	type fields struct {
		TLS                  *TLS
		Role                 string
		Cluster              string
		CAdvisorURL          string
		CAdvisorMetricFilter []string
		CAdvisorDataSendRate int
	}
	type args struct {
		testData string
		cadvisor map[string]interface{}
	}
	tests := []struct {
		name             string
		fields           fields
		expectedCAdvisor map[string]interface{}
		args             args
		wantErr          bool
	}{
		{
			"Kubernetes.ParseCAdvisor() valid worker",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/cert",
					ClientKey:  "/path/to/key",
					CACert:     "/path/to/ca",
				},
				Role:        "worker",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:4493",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 29,
			},
			map[string]interface{}{
				"excludedMetrics": map[string]bool{
					"container_cpu_utilization":          true,
					"container_cpu_utilization_per_core": true,
				},
				"cadvisorurl":  "http://localhost:4493",
				"dataSendRate": 29,
			},
			args{
				testData: "testdata/kubernetes/kubernetes-valid-worker.yaml",
				cadvisor: map[string]interface{}{},
			},
			false,
		},
		{
			"Kubernetes.ParseCAdvisor() valid master",
			fields{
				TLS: &TLS{
					SkipVerify: true,
					ClientCert: "/path/to/certAlt",
					ClientKey:  "/path/to/keyAlt",
					CACert:     "/path/to/caAlt",
				},
				Role:        "worker",
				Cluster:     "kubernetes-cluster",
				CAdvisorURL: "http://localhost:8080",
				CAdvisorMetricFilter: []string{
					"container_cpu_utilization",
					"container_cpu_utilization_per_core",
				},
				CAdvisorDataSendRate: 30,
			},
			map[string]interface{}{
				"excludedMetrics": map[string]bool{
					"container_cpu_utilization":          true,
					"container_cpu_utilization_per_core": true,
				},
				"cadvisorurl":  "http://localhost:8080",
				"dataSendRate": 30,
			},
			args{
				testData: "testdata/kubernetes/kubernetes-valid-alt-worker.yaml",
				cadvisor: map[string]interface{}{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Kubernetes{
				TLS:                  tt.fields.TLS,
				Role:                 tt.fields.Role,
				Cluster:              tt.fields.Cluster,
				CAdvisorURL:          tt.fields.CAdvisorURL,
				CAdvisorMetricFilter: tt.fields.CAdvisorMetricFilter,
				CAdvisorDataSendRate: tt.fields.CAdvisorDataSendRate,
			}

			var kubernetes = &Kubernetes{}
			var err error
			if err = kubernetes.LoadYAML(tt.args.testData); err == nil {
				if !reflect.DeepEqual(kubernetes, k) {
					pretty.Ldiff(t, kubernetes, k)
					t.Error("Kubernetes.ParseCadvisor() Differences detected")
				}
			}

			if err := kubernetes.ParseCAdvisor(tt.args.cadvisor); (err != nil) != tt.wantErr {
				t.Errorf("Kubernetes.ParseCadvisor() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.expectedCAdvisor, tt.args.cadvisor) {
				pretty.Ldiff(t, tt.expectedCAdvisor, tt.args.cadvisor)
				t.Error("Kubernetes.ParseCadvisor() Differences detected")
			}
		})
	}
}
