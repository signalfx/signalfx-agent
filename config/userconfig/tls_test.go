package userconfig

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestTLS_Parse(t *testing.T) {
	type fields struct {
		SkipVerify bool
		ClientCert string
		ClientKey  string
		CACert     string
	}
	type args struct {
		testData string
		tls      map[string]interface{}
	}
	tests := []struct {
		name        string
		fields      fields
		expectedTLS map[string]interface{}
		args        args
		wantErr     bool
	}{
		{
			"TLS.Parse()",
			fields{
				SkipVerify: true,
				ClientCert: "/path/to/cert",
				ClientKey:  "/path/to/key",
				CACert:     "/path/to/ca",
			},
			map[string]interface{}{
				"caCert":     "/path/to/ca",
				"clientKey":  "/path/to/key",
				"clientCert": "/path/to/cert",
				"skipVerify": true,
			},
			args{
				testData: "testdata/tls/tls.yaml",
				tls:      map[string]interface{}{},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := &TLS{
				SkipVerify: tt.fields.SkipVerify,
				ClientCert: tt.fields.ClientCert,
				ClientKey:  tt.fields.ClientKey,
				CACert:     tt.fields.CACert,
			}

			var tls = &TLS{}
			var err error
			if err = tls.LoadYAML(tt.args.testData); err == nil {
				if !reflect.DeepEqual(tls, tl) {
					pretty.Ldiff(t, tls, tl)
					t.Error("TLS.Parse() Differences detected")
				}
			}

			if err := tls.Parse(tt.args.tls); (err != nil) != tt.wantErr {
				t.Errorf("TLS.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.expectedTLS, tt.args.tls) {
				pretty.Ldiff(t, tt.expectedTLS, tt.args.tls)
				t.Error("Proxy.Parse() Differences detected")
			}
		})
	}
}
