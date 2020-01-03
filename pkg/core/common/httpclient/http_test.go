package httpclient

import (
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestHttpConfig_Build(t *testing.T) {
	type fields struct {
		HTTPTimeout    time.Duration
		Username       string
		Password       string
		UseHTTPS       bool
		SkipVerify     bool
		CACertPath     string
		ClientCertPath string
		ClientKeyPath  string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *http.Client
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTPConfig{
				HTTPTimeout:    tt.fields.HTTPTimeout,
				Username:       tt.fields.Username,
				Password:       tt.fields.Password,
				UseHTTPS:       tt.fields.UseHTTPS,
				SkipVerify:     tt.fields.SkipVerify,
				CACertPath:     tt.fields.CACertPath,
				ClientCertPath: tt.fields.ClientCertPath,
				ClientKeyPath:  tt.fields.ClientKeyPath,
			}
			got, err := h.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHttpConfig_Scheme(t *testing.T) {
	type fields struct {
		UseHTTPS       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"https enabled", fields{UseHTTPS:true}, "https"},
		{"https disabled", fields{UseHTTPS:false}, "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTPConfig{
				UseHTTPS:       tt.fields.UseHTTPS,
			}
			if got := h.Scheme(); got != tt.want {
				t.Errorf("Scheme() = %v, want %v", got, tt.want)
			}
		})
	}
}
