package userconfig

import (
	"reflect"
	"testing"

	"github.com/kr/pretty"
)

func TestFilter_Parse(t *testing.T) {
	type fields struct {
		DockerContainerNames     []string
		Images                   []string
		KubernetesContainerNames []string
		KubernetesPodNames       []string
		KubernetesNamespaces     []string
		Labels                   []*Label
	}
	type args struct {
		store map[string]interface{}
		file  string
	}
	tests := []struct {
		name    string
		fields  fields
		parsed  map[string]interface{}
		args    args
		wantErr bool
	}{
		{
			"Filter.Parse() load config",
			fields{
				[]string{ // dockerContainerNames
					"^somethingsnarky$",
				},
				[]string{
					"^elasticsearch:*",
					"^mysql:*",
				},
				[]string{
					"^redis:*",
				},
				[]string{
					"^podNameOne$",
				},
				[]string{
					"^default$",
				},
				[]*Label{
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
			map[string]interface{}{
				"excludedImages": []string{
					"^elasticsearch:*",
					"^mysql:*",
				},
				"excludedNames": []string{
					"^somethingsnarky$",
				},
				"excludedLabels": [][]string{
					[]string{
						"^environment$",
						"^dev$",
					},
					[]string{
						"^environment$",
						"^test$",
					},
					[]string{
						"^io.kubernetes.pod.namespace$",
						"^default$",
					},
					[]string{
						"^io.kubernetes.container.name$",
						"^redis:*",
					},
					[]string{
						"^io.kubernetes.pod.name$",
						"^podNameOne$",
					},
				},
			},
			args{
				store: map[string]interface{}{},
				file:  "testdata/filter/filter.yaml",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Filter{
				DockerContainerNames: tt.fields.DockerContainerNames,
				Images:               tt.fields.Images,
				KubernetesContainerNames: tt.fields.KubernetesContainerNames,
				KubernetesPodNames:       tt.fields.KubernetesPodNames,
				KubernetesNamespaces:     tt.fields.KubernetesNamespaces,
				Labels:                   tt.fields.Labels,
			}

			var filter = &Filter{}
			var err error
			if err = filter.LoadYAML(tt.args.file); err == nil {
				if !reflect.DeepEqual(filter, f) {
					pretty.Ldiff(t, filter, f)
					t.Error("Filter.Parse() Differences detected")
				}
			}

			if err := filter.Parse(tt.args.store); (err != nil) != tt.wantErr {
				t.Errorf("Filter.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.parsed, tt.args.store) {
				pretty.Ldiff(t, tt.parsed, tt.args.store)
				t.Error("Filter.Parse() Differences detected")
			}
		})
	}
}

func TestFilter_GetLabels(t *testing.T) {
	type fields struct {
		DockerContainerNames     []string
		Images                   []string
		KubernetesContainerNames []string
		KubernetesPodNames       []string
		KubernetesNamespaces     []string
		Labels                   []*Label
	}
	type args struct {
		file string
	}
	tests := []struct {
		name   string
		fields fields
		want   []*Label
		args   args
	}{
		{
			"Filter.GetLabels() load labels",
			fields{
				[]string{ // dockerContainerNames
					"^somethingsnarky$",
				},
				[]string{
					"^elasticsearch:*",
					"^mysql:*",
				},
				[]string{
					"^redis:*",
				},
				[]string{
					"^podNameOne$",
				},
				[]string{
					"^default$",
				},
				[]*Label{
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
			[]*Label{
				&Label{
					Key:   "^environment$",
					Value: "^dev$",
				},
				&Label{
					Key:   "^environment$",
					Value: "^test$",
				},
				&Label{
					Key:   "^io.kubernetes.pod.namespace$",
					Value: "^default$",
				},
				&Label{
					Key:   "^io.kubernetes.container.name$",
					Value: "^redis:*",
				},
				&Label{
					Key:   "^io.kubernetes.pod.name$",
					Value: "^podNameOne$",
				},
			},
			args{
				file: "testdata/filter/filter.yaml",
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Filter{
				DockerContainerNames: tt.fields.DockerContainerNames,
				Images:               tt.fields.Images,
				KubernetesContainerNames: tt.fields.KubernetesContainerNames,
				KubernetesPodNames:       tt.fields.KubernetesPodNames,
				KubernetesNamespaces:     tt.fields.KubernetesNamespaces,
				Labels:                   tt.fields.Labels,
			}

			var filter = &Filter{}
			var err error
			if err = filter.LoadYAML(tt.args.file); err == nil {
				if !reflect.DeepEqual(filter, f) {
					pretty.Ldiff(t, filter, f)
					t.Error("Filter.Parse() Differences detected")
				}
			}

			if got := f.GetLabels(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter.GetLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
