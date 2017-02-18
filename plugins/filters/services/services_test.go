package services

import (
	"reflect"
	"testing"

	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

// May want to share these fixtures in the future with new tests.
var discoveredApache = services.Instance{
	ID: "test-instance",
	Service: &services.Service{
		Name: "default",
	},
	Container: &services.Container{
		Names:  []string{"apache"},
		Image:  "apache",
		Labels: map[string]string{},
	},
	Port: &services.Port{
		IP:          "localhost",
		Type:        services.TCP,
		PrivatePort: 80,
		PublicPort:  8080,
	},
}

var discoveredRedis = services.Instance{
	ID: "test-instance",
	Service: &services.Service{
		Name: "default",
	},
	Container: &services.Container{
		Names:  []string{"redis"},
		Image:  "unknown",
		Labels: map[string]string{"signalfx-integration": "redis"},
	},
	Port: &services.Port{
		IP:          "localhost",
		Type:        services.TCP,
		PrivatePort: 3689,
		PublicPort:  3689,
	},
}

// Service instance that has been mapped to Apache service type.
var apacheImageMapped = discoveredApache

// Service instance that has been mapped to a custom name.
var apacheCustom = discoveredApache

// Service instance that has been mapped based on label.
var redisLabelMapped = discoveredRedis

func init() {
	// Customize cloned service instances.
	apacheImageMapped.Service = &services.Service{
		Name: "apache-image-default",
		Type: services.ApacheService,
	}
	apacheCustom.Service = &services.Service{
		Name: "apache-custom",
		Type: services.ApacheService,
	}

	redisLabelMapped.Service = &services.Service{
		Name: "redis-labeled-default",
		Type: services.RedisService,
	}
}

func TestNewRuleFilter(t *testing.T) {
	configPresent := viper.New()
	configPresent.Set("servicesfiles", []string{"testdata/defaults.json"})

	configMissing := viper.New()

	type args struct {
		name   string
		config *viper.Viper
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"configuration missing", args{"services", configMissing}, true},
		{"configuration present", args{"services", configPresent}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRuleFilter(tt.args.name, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRuleFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRuleFilter_Map(t *testing.T) {
	// Has to be an empty slice and not just nil for the DeepEqual comparison to work.
	sis := make(services.Instances, 0)

	makeFilter := func(files ...string) *RuleFilter {
		v := viper.New()
		v.Set("servicesfiles", files)

		filter, err := NewRuleFilter("rules", v)
		if err != nil {
			t.Errorf("NewRuleFilter() failed: %s", err)
		}
		return filter
	}

	emptyRules := makeFilter("testdata/zero-signatures.json")
	customRules := makeFilter("testdata/custom.json", "testdata/defaults.json")
	defaultRules := makeFilter("testdata/defaults.json")

	type args struct {
		sis services.Instances
	}
	tests := []struct {
		name    string
		filter  *RuleFilter
		args    args
		want    services.Instances
		wantErr bool
	}{
		{"empty rules", emptyRules, args{sis}, sis, false},
		{"custom rule wins", customRules, args{services.Instances{discoveredApache}},
			services.Instances{apacheCustom}, false},
		{"default rules match", defaultRules, args{services.Instances{discoveredApache, discoveredRedis}},
			services.Instances{apacheImageMapped, redisLabelMapped}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.filter.Map(tt.args.sis)
			if (err != nil) != tt.wantErr {
				t.Errorf("RuleFilter.Map() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RuleFilter.Map() = %v, want %v", got, tt.want)
			}
		})
	}
}
