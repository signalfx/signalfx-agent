package baseemitter

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
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
		excludeTag         []string
		addTag             map[string]string
	}
	ts := time.Now()
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *testOutput
	}{
		{
			name: "emit datapoint without plugin tag",
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
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key": "dim1Val",
							"plugin":  "name",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit datapoint with plugin tag",
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
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key": "dim1Val",
							"plugin":  "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit datapoint with metric type that defaults to gauge",
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
				originalMetricType: "untyped",
				t:                  []time.Time{ts},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"telegraf_type": "untyped",
							"dim1Key":       "dim1Val",
							"plugin":        "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit datapoint and remove an undesirable tag",
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
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
				excludeTag: []string{"dim1Key"},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"plugin": "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit datapoint and add a tag",
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
					"plugin": "pluginname",
				},
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
				addTag:     map[string]string{"dim1Key": "dim1Val"},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key": "dim1Val",
							"plugin":  "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit datapoint and add a tag that overrides an original tag",
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
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
				addTag:     map[string]string{"dim1Key": "dim1Override"},
			},
			want: &testOutput{
				Datapoints: []*datapoint.Datapoint{
					datapoint.New(
						"name.fieldname",
						map[string]string{
							"dim1Key": "dim1Override",
							"plugin":  "pluginname",
						},
						datapoint.NewIntValue(int64(5)),
						datapoint.Gauge,
						ts),
				},
			},
		},
		{
			name: "emit an event",
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
				metricType: datapoint.Gauge,
				t:          []time.Time{ts},
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
							"dim1Key": "dim1Val",
							"plugin":  "name",
						},
						map[string]interface{}{
							"message": "hello world",
						},
						ts),
				},
			},
		},
		{
			name: "exclude events that are not explicitly included or sf_metrics",
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
				metricType:   datapoint.Gauge,
				t:            []time.Time{ts},
				includeEvent: []string{},
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
				metricType:   datapoint.Gauge,
				t:            []time.Time{ts},
				includeEvent: []string{},
				excludeData:  []string{"name.fieldname"},
			},
			want: &testOutput{
				Events:     []*event.Event{},
				Datapoints: []*datapoint.Datapoint{},
			},
		},
		{
			name: "malformed property objects should be dropped",
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
				metricType:   datapoint.Gauge,
				t:            []time.Time{ts},
				includeEvent: []string{},
				excludeData:  []string{"name.fieldname"},
			},
			want: &testOutput{
				Events:     []*event.Event{},
				Datapoints: []*datapoint.Datapoint{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			I := NewEmitter(tt.fields.Output, log.NewEntry(log.New()))
			I.AddTags(tt.args.addTag)
			I.IncludeEvents(tt.args.includeEvent)
			I.ExcludeData(tt.args.excludeData)
			I.OmitTags(tt.args.excludeTag)
			I.Add(tt.args.measurement, tt.args.fields, tt.args.tags,
				tt.args.metricType, tt.args.originalMetricType, tt.args.t...)

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

func Test_GetTime(t *testing.T) {
	t.Run("current time is returned when no time is supplied", func(t *testing.T) {
		if got := GetTime([]time.Time{}...); got.Unix() < 1 {
			t.Error("GetTime() did not return a time")
		}
	})
}

func TestBaseEmitter_AddError(t *testing.T) {
	t.Run("test", func(t *testing.T) {
		var buffer bytes.Buffer
		logger := log.New()
		logger.Out = &buffer
		entry := log.NewEntry(logger)
		err := fmt.Errorf("errorz test")
		B := &BaseEmitter{
			Logger: entry,
		}
		B.AddError(err)
		if !strings.Contains(buffer.String(), "errorz test") {
			t.Errorf("AddError() expected error string with 'errorz test' but got '%s'", buffer.String())
		}
	})
}
