package integrations

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/kr/pretty"
	. "github.com/signalfx/neo-agent/neotest"
	"github.com/signalfx/neo-agent/pipelines"
	"github.com/signalfx/neo-agent/services"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
)

func init() {
	// Loggig makes test output hard to read.
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

var discovered = services.Instances{
	services.Instance{
		ID: "redis-abcd",
		Service: &services.Service{
			Name: "redis-abcd",
		},
		Container: &services.Container{
			Names:  []string{"redis"},
			Image:  "redis:3.2.4",
			Labels: map[string]string{"app": "redis"},
		},
		Port: &services.Port{
			IP:          "192.168.1.1",
			Type:        services.TCP,
			PrivatePort: 6379,
			PublicPort:  6379,
		},
	}, services.Instance{
		ID: "redis-efgh",
		Service: &services.Service{
			Name: "redis-efgh",
		},
		Container: &services.Container{
			Names:  []string{"redis"},
			Image:  "redis:3.2.4",
			Labels: map[string]string{"app": "redis", "env": "production"},
		},
		Port: &services.Port{
			IP:          "192.168.1.2",
			Type:        services.TCP,
			PrivatePort: 6379,
			PublicPort:  6379,
		},
	}, services.Instance{
		ID: "redis-ijkl",
		Service: &services.Service{
			Name: "redis-ijkl",
		},
		Container: &services.Container{
			Names:  []string{"redis"},
			Image:  "redis:3.2.4",
			Labels: map[string]string{"app": "redis", "env": "production"},
		},
		Port: &services.Port{
			IP:          "192.168.1.3",
			Type:        services.TCP,
			PrivatePort: 6379,
			PublicPort:  6379,
		},
	}, services.Instance{
		ID: "other-abcd",
		Service: &services.Service{
			Name: "other-abcd",
		},
		Container: &services.Container{
			Names:  []string{"redis"},
			Image:  "other:3.5.0",
			Labels: map[string]string{"app": "other", "env": "production"},
		},
		Port: &services.Port{
			IP:          "192.168.1.4",
			Type:        services.TCP,
			PrivatePort: 3689,
			PublicPort:  3689,
		},
	},
}

type InstancePatch struct {
	Type     services.ServiceType
	template string
	vars     map[string]interface{}
}

func patchInstances(patches []InstancePatch) services.Instances {
	var mapped services.Instances

	for i, patch := range patches {
		if patch.Type == services.UnknownService {
			continue
		}
		inst := discovered[i]
		inst.Service.Type = patch.Type
		inst.Template = patch.template
		inst.Vars = patch.vars
		mapped = append(mapped, inst)
	}

	return mapped
}

func TestTemplates(t *testing.T) {
	builtin := `
    integrations:
        redis:
            rule: true
            template:
`

	override := `
    integrations:
        redis:
            configurations:
`

	builtinConfig, err := loadConfigs([][]byte{[]byte(builtin)})
	if err != nil {
		t.Fatal(err)
	}

	overrideConfig, err := loadConfigs([][]byte{[]byte(override)})
	if err != nil {
		t.Fatal(err)
	}

	// Test the cross product of the 3 levels (builtin, integration,
	// configuration) and the template being set or unset.
	type setType struct {
		builtin       string
		integration   string
		configuration []string
		wantErr       bool
		want          string
	}
	tests := []struct {
		name string
		sets []setType
	}{
		{"one configuration", []setType{
			{"", "", []string{""}, true, ""},
			{"a", "", []string{""}, false, "a"},
			{"a", "b", []string{""}, false, "b"},
			{"a", "b", []string{"c"}, false, "c"},
			{"", "b", []string{""}, false, "b"},
			{"", "b", []string{"c"}, false, "c"},
			{"", "", []string{"c"}, false, "c"},
			{"a", "", []string{"c"}, false, "c"},
		},
		},
		{"zero configurations", []setType{
			{"", "", nil, true, ""},
			{"a", "", nil, false, "a"},
			{"a", "b", nil, false, "b"},
			{"", "b", nil, false, "b"},
		},
		},
	}

	Convey("Test template propagation", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				for _, set := range tt.sets {
					Convey(fmt.Sprintf("%+v", set), func() {
						builtinConfig[0].Integrations["redis"].Template = set.builtin
						overrideConfig[0].Integrations["redis"].Template = set.integration
						overrideConfig[0].Integrations["redis"].Configurations = map[string]*integConfig{}

						for i, c := range set.configuration {
							overrideConfig[0].Integrations["redis"].Configurations[strconv.Itoa(i)] = &integConfig{Template: c, Rule: "true"}
						}

						got, err := buildConfigurations(builtinConfig, overrideConfig)
						if set.wantErr {
							So(err, ShouldNotBeNil)
						} else {
							So(err, ShouldBeNil)
							So(got, ShouldHaveLength, 1)
							So(got[0].template, ShouldEqual, set.want)
						}
					})
				}
			})
		}
	})
}

func TestRules(t *testing.T) {
	builtin := `
    integrations:
        redis:
            rule: true
            template: template
`

	override := `
    integrations:
        redis:
            configurations:
`

	builtinConfig, err := loadConfigs([][]byte{[]byte(builtin)})
	if err != nil {
		t.Fatal(err)
	}

	overrideConfig, err := loadConfigs([][]byte{[]byte(override)})
	if err != nil {
		t.Fatal(err)
	}

	// Test the cross product of the 3 levels (builtin, integration,
	// configuration) and the template being set or unset.
	type setType struct {
		builtin       string
		integration   string
		configuration []string
		wantErr       bool
		want          string
	}
	tests := []struct {
		name string
		sets []setType
	}{
		{"one configuration", []setType{
			{"", "", []string{""}, true, ""},
			{"1", "", []string{""}, true, ""},
			{"1", "2", []string{""}, true, ""},
			{"1", "true", []string{"false"}, false, "(true) && (false)"},
			{"", "2", []string{""}, true, ""},
			{"", "true", []string{"false"}, false, "(true) && (false)"},
			{"", "", []string{"3"}, false, "3"},
			{"1", "", []string{"3"}, false, "3"},
		},
		},
		{"zero configurations", []setType{
			{"", "", nil, true, ""},
			{"1", "", nil, false, "1"},
			{"1", "2", nil, false, "2"},
			{"", "2", nil, false, "2"},
		},
		},
	}

	Convey("Test rule propagation", t, func() {
		for _, tt := range tests {
			Convey(tt.name, func() {
				for _, set := range tt.sets {
					Convey(fmt.Sprintf("%+v", set), func() {
						builtinConfig[0].Integrations["redis"].Rule = set.builtin
						overrideConfig[0].Integrations["redis"].Rule = set.integration
						overrideConfig[0].Integrations["redis"].Configurations = map[string]*integConfig{}

						for i, c := range set.configuration {
							overrideConfig[0].Integrations["redis"].Configurations[strconv.Itoa(i)] = &integConfig{Rule: c, Template: "template"}
						}

						got, err := buildConfigurations(builtinConfig, overrideConfig)
						if set.wantErr {
							So(err, ShouldNotBeNil)
						} else {
							So(err, ShouldBeNil)
							So(got, ShouldHaveLength, 1)
							So(got[0].ruleText, ShouldEqual, set.want)
						}
					})
				}
			})
		}
	})
}

func TestDisabled(t *testing.T) {
	config := viper.New()
	config.Set("builtins", "testdata/builtins")
	config.Set("overrides", "testdata/redis-disabled")
	filter, err := NewFilter("filter", config)
	if err != nil {
		t.Fatal(err)
	}
	f := filter.(*Filter)
	Must(t, f.load(config))
	Convey("Test disabling builtins", t, func() {
		So(f.configurations, ShouldHaveLength, 1)
		So(f.configurations[0].serviceType, ShouldEqual, services.ApacheService)
	})
}

func TestVariables(t *testing.T) {
	Must(t, os.Setenv("REDIS_AUTH", "password"))
	defer func() { Must(t, os.Unsetenv("REDIS_AUTH")) }()

	config := viper.New()
	config.Set("builtins", "testdata/builtins")
	config.Set("overrides", "testdata/config-var-template")
	filter, err := NewFilter("filter", config)
	if err != nil {
		t.Fatal(err)
	}
	f := filter.(*Filter)

	Convey("Variables are resolved", t, func() {
		So(f.configurations, ShouldHaveLength, 2)

		for _, c := range f.configurations {
			if c.serviceType == services.RedisService {
				So(c.vars, ShouldResemble, map[string]interface{}{"Auth": "password", "Username": "static"})
				return
			}
		}

		t.Errorf("redis integration not found")
	})
}

func TestMap(t *testing.T) {
	s1 := []InstancePatch{
		{services.RedisService, "override-template", map[string]interface{}{"Auth": "password"}},
		{services.RedisService, "override-template", map[string]interface{}{"Auth": "password"}},
		{services.RedisService, "override-template", map[string]interface{}{"Auth": "password"}},
	}

	s2 := []InstancePatch{
		{},
		{services.RedisService, "production-template", map[string]interface{}{}},
		{services.RedisService, "production-template", map[string]interface{}{}},
		{},
	}

	type args struct {
		services services.Instances
	}
	tests := []struct {
		name      string
		args      args
		builtins  string
		overrides string
		patches   []InstancePatch
		wantErr   bool
	}{
		{"basic integration without configurations", args{discovered}, "testdata/builtins", "testdata/basic", s1, false},
		{"configuration", args{discovered}, "testdata/builtins", "testdata/configurations", s2, false},
	}
	for _, tt := range tests {
		config := viper.New()
		config.Set("builtins", tt.builtins)
		config.Set("overrides", tt.overrides)
		filter, err := NewFilter("filter", config)
		if err != nil {
			t.Fatal(err)
		}
		ss := filter.(pipelines.SourceSink)

		t.Run(tt.name, func(t *testing.T) {
			got, err := ss.Map(tt.args.services)
			want := patchInstances(tt.patches)
			if (err != nil) != tt.wantErr {
				pretty.Ldiff(t, got, want)
				t.Error("differences detected")
				return
			}
			if !reflect.DeepEqual(got, want) {
				pretty.Ldiff(t, got, want)
				t.Error("differences detected")
			}
		})
	}
}
