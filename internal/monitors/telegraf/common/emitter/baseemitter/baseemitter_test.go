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
		excludeTag         []string
		addTag             map[string]string
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
		{
			name: "emit datapoint with plugin tag",
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
			wantDatapoints: []*datapoint.Datapoint{
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
		{
			name: "emit datapoint with metric type that defaults to gauge",
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
			wantDatapoints: []*datapoint.Datapoint{
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
		{
			name: "emit datapoint and remove an undesirable tag",
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
			wantDatapoints: []*datapoint.Datapoint{
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
		{
			name: "emit datapoint and add a tag",
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
			wantDatapoints: []*datapoint.Datapoint{
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
		{
			name: "emit datapoint and add a tag that overrides an original tag",
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
			wantDatapoints: []*datapoint.Datapoint{
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
		{
			name: "emit an event",
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
			wantEvents: []*event.Event{
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
		{
			name: "exclude events that are not explicitly included or sf_metrics",
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
			wantEvents: nil,
		},
		{
			name: "exclude data by metric name",
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
			wantEvents:     nil,
			wantDatapoints: nil,
		},
		{
			name: "malformed property objects should be dropped",
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
			wantEvents:     nil,
			wantDatapoints: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := neotest.NewTestOutput()
			lg := log.NewEntry(log.New())
			I := NewEmitter(out, lg)
			I.AddTags(tt.args.addTag)
			I.IncludeEvents(tt.args.includeEvent)
			I.ExcludeData(tt.args.excludeData)
			I.OmitTags(tt.args.excludeTag)
			I.Add(tt.args.measurement, tt.args.fields, tt.args.tags,
				tt.args.metricType, tt.args.originalMetricType, tt.args.t...)

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
