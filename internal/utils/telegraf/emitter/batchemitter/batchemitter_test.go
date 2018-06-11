package batchemitter

import (
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

type testOutput struct {
	Datapoints []*datapoint.Datapoint
	Events     []*event.Event
	Props      []*types.DimProperties
}

func (to *testOutput) SendDatapoint(dp *datapoint.Datapoint) {
	to.Datapoints = append(to.Datapoints, dp)
}

func (to *testOutput) SendEvent(ev *event.Event) {
	to.Events = append(to.Events, ev)
}

func (to *testOutput) SendDimensionProps(prop *types.DimProperties) {
	to.Props = append(to.Props, prop)
}

func TestImmediateEmitter_Emit(t *testing.T) {
	type fields struct {
		Output types.Output
	}
	type args struct {
		measurement        string
		fields             map[string]interface{}
		tags               map[string]string
		metricType         datapoint.MetricType
		originalMetricType string
		t                  []time.Time
		includeEvent       []string
		excludeData        []string
	}
	ts := time.Now()
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *testOutput
	}{
		{
			name: "single datapoint with timestamp and no plugin dimension",
			fields: fields{
				Output: &testOutput{
					Datapoints: []*datapoint.Datapoint{},
				},
			},
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": 5,
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key":       "dim1Val",
							"plugin":        "name",
							"telegraf_type": "gauge",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "single datapoint with plugin dimension",
			fields: fields{
				Output: &testOutput{
					Datapoints: []*datapoint.Datapoint{},
				},
			},
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": 5,
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
					"plugin":  "pluginname",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key":       "dim1Val",
							"telegraf_type": "gauge",
							"plugin":        "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "single event with timestamp and no plugin dimension",
			fields: fields{
				Output: &testOutput{
					Events: []*event.Event{},
				},
			},
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": "hello world",
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
				includeEvent: []string{
					"name.fieldname",
				},
			},
			want: &testOutput{
				Events: []*event.Event{
					event.NewWithProperties(
						"name.fieldname",
						event.AGENT,
						map[string]string{
							"dim1Key":       "dim1Val",
							"telegraf_type": "gauge",
							"plugin":        "name",
						},
						map[string]interface{}{
							"message": "hello world",
						},
						ts),
				},
			},
		},
		{
			name: "single excluded event",
			fields: fields{
				Output: &testOutput{
					Events: []*event.Event{},
				},
			},
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": "hello world",
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
					"plugin":  "pluginname",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
				includeEvent:       []string{},
			},
			want: &testOutput{
				Events: []*event.Event{},
			},
		},
		{
			name: "exclude data by metric name",
			fields: fields{
				Output: &testOutput{
					Events:     []*event.Event{},
					Datapoints: []*datapoint.Datapoint{},
				},
			},
			args: args{
				measurement: "name",
				fields: map[string]interface{}{
					"fieldname": 5,
				},
				tags: map[string]string{
					"dim1Key": "dim1Val",
					"plugin":  "pluginname",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
				includeEvent:       []string{},
				excludeData:        []string{"name.fieldname"},
			},
			want: &testOutput{
				Events:     []*event.Event{},
				Datapoints: []*datapoint.Datapoint{},
			},
		},
		{
			name: "malformed property object",
			fields: fields{
				Output: &testOutput{
					Events:     []*event.Event{},
					Datapoints: []*datapoint.Datapoint{},
				},
			},
			args: args{
				measurement: "objects",
				fields: map[string]interface{}{
					"value": "",
				},
				tags: map[string]string{
					"sf_metric": "objects.host-meta-data",
					"plugin":    "signalfx-metadata",
					"severity":  "4",
				},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  []time.Time{ts},
				includeEvent:       []string{},
				excludeData:        []string{"name.fieldname"},
			},
			want: &testOutput{
				Events:     []*event.Event{},
				Datapoints: []*datapoint.Datapoint{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			I := NewEmitter()
			I.Logger = log.NewEntry(log.New())
			I.Output = tt.fields.Output
			I.IncludeEvents(tt.args.includeEvent)
			I.ExcludeData(tt.args.excludeData)
			I.Add(tt.args.measurement, tt.args.fields, tt.args.tags,
				tt.args.metricType, tt.args.originalMetricType, tt.args.t...)
			I.Send()
			if !reflect.DeepEqual(I.Output.(*testOutput), tt.want) {
				t.Errorf("actual output: %v does not match desired: %v", I.Output.(*testOutput), tt.want)
				if len(I.Output.(*testOutput).Datapoints) != len(tt.want.Datapoints) {
					t.Errorf("length of output datapoints (%d) != desired len (%d)", len(I.Output.(*testOutput).Datapoints), len(tt.want.Datapoints))
				}
				if len(I.Output.(*testOutput).Events) != len(tt.want.Events) {
					t.Errorf("length of out events (%d) != desired len (%d)", len(I.Output.(*testOutput).Events), len(tt.want.Events))
				}
				if len(I.Output.(*testOutput).Props) != len(tt.want.Props) {
					t.Errorf("length of out events (%d) != desired len (%d)", len(I.Output.(*testOutput).Props), len(tt.want.Props))
				}
			}
		})
	}
}
