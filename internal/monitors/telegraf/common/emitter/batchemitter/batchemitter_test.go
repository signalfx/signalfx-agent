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
