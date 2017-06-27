package userconfig

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestProxy_Parse(t *testing.T) {
	type fields struct {
		HTTP  string
		HTTPS string
		Skip  string
	}
	type args struct {
		testData string
		proxy    map[string]string
	}
	tests := []struct {
		name          string
		fields        fields
		expectedProxy map[string]string
		args          args
		wantErr       bool
	}{
		{
			"Proxy.Parse()",
			fields{
				HTTP:  "http://www.notarealproxy.com",
				HTTPS: "https://www.notarealproxy.com",
				Skip:  "http://www.skipthis.com",
			},
			map[string]string{
				"http":  "http://www.notarealproxy.com",
				"https": "https://www.notarealproxy.com",
				"skip":  "http://www.skipthis.com",
			},
			args{
				testData: "testdata/proxy/proxy.yaml",
				proxy:    map[string]string{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Proxy{
				HTTP:  tt.fields.HTTP,
				HTTPS: tt.fields.HTTPS,
				Skip:  tt.fields.Skip,
			}

			var proxy = &Proxy{}
			var err error
			if err = proxy.LoadYAML(tt.args.testData); err == nil {
				if !reflect.DeepEqual(proxy, p) {
					pretty.Ldiff(t, proxy, p)
					t.Error("Proxy.Parse() Differences detected")
				}
			}

			if err := proxy.Parse(tt.args.proxy); (err != nil) != tt.wantErr {
				t.Errorf("Proxy.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.expectedProxy, tt.args.proxy) {
				pretty.Ldiff(t, tt.expectedProxy, tt.args.proxy)
				t.Error("Proxy.Parse() Differences detected")
			}
		})
	}
}
