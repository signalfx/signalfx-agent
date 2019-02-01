package batchemitter

import (
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/signalfx-agent/internal/neotest"
	log "github.com/sirupsen/logrus"
)

func TestImmediateEmitter_Emit(t *testing.T) {
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
		name           string
		args           args
		wantDatapoints []*datapoint.Datapoint
		wantEvents     []*event.Event
	}{
		{
			name: "emit datapoint without plugin tag",
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
			wantDatapoints: []*datapoint.Datapoint{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := neotest.NewTestOutput()
			lg := log.NewEntry(log.New())
			I := NewEmitter(out, lg)
			I.IncludeEvents(tt.args.includeEvent)
			I.ExcludeData(tt.args.excludeData)
			I.Add(tt.args.measurement, tt.args.fields, tt.args.tags,
				tt.args.metricType, tt.args.originalMetricType, tt.args.t...)
			I.Send()

			dps := out.FlushDatapoints()
			if !reflect.DeepEqual(dps, tt.wantDatapoints) {
				t.Errorf("actual output: datapoints %v does not match desired: %v", dps, tt.wantDatapoints)
			}

			events := out.FlushEvents()
			if !reflect.DeepEqual(events, tt.wantEvents) {
				t.Errorf("actual output: events %v does not match desired: %v", dps, tt.wantDatapoints)
			}
		})
	}
}
