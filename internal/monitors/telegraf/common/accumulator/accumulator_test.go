package accumulator

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
)

type testEmitter struct {
	measurement        string
	fields             map[string]interface{}
	tags               map[string]string
	metricType         datapoint.MetricType
	originalMetricType string
	t                  time.Time
	err                error
}

func (e *testEmitter) Add(measurement string, fields map[string]interface{},
	tags map[string]string, metricType datapoint.MetricType, originalMetricType string, t ...time.Time) {
	e.measurement = measurement
	e.fields = fields
	e.tags = tags
	e.metricType = metricType
	e.originalMetricType = originalMetricType
	e.t = t[0]
}
func (e *testEmitter) IncludeEvent(string)       {}
func (e *testEmitter) IncludeEvents([]string)    {}
func (e *testEmitter) ExcludeDatum(string)       {}
func (e *testEmitter) ExcludeData([]string)      {}
func (e *testEmitter) AddTag(string, string)     {}
func (e *testEmitter) AddTags(map[string]string) {}
func (e *testEmitter) OmitTag(string)            {}
func (e *testEmitter) OmitTags([]string)         {}
func (e *testEmitter) AddError(err error) {
	e.err = err
}

func TestAccumulator(t *testing.T) {
	ac := NewAccumulator(&testEmitter{})
	tests := []struct {
		name string
		want *testEmitter
		fn   func(string, map[string]interface{}, map[string]string, ...time.Time)
	}{
		{
			name: "AddFields()",
			want: &testEmitter{
				measurement:        "field_measurement",
				fields:             map[string]interface{}{"dim1": "dimval1"},
				tags:               map[string]string{"tag1": "tagval1"},
				metricType:         datapoint.Gauge,
				originalMetricType: "untyped",
				t:                  time.Now(),
			},
			fn: ac.AddFields,
		},
		{
			name: "AddGauge()",
			want: &testEmitter{
				measurement:        "gauge_measurement",
				fields:             map[string]interface{}{"dim1": "dimval1"},
				tags:               map[string]string{"tag1": "tagval1"},
				metricType:         datapoint.Gauge,
				originalMetricType: "gauge",
				t:                  time.Now(),
			},
			fn: ac.AddGauge,
		},
		{
			name: "AddCounter()",
			want: &testEmitter{
				measurement:        "counter_measurement",
				fields:             map[string]interface{}{"dim1": "dimval1"},
				tags:               map[string]string{"tag1": "tagval1"},
				metricType:         datapoint.Counter,
				originalMetricType: "counter",
				t:                  time.Now(),
			},
			fn: ac.AddCounter,
		},
		{
			name: "AddSummary()",
			want: &testEmitter{
				measurement:        "summary_measurement",
				fields:             map[string]interface{}{"dim1": "dimval1"},
				tags:               map[string]string{"tag1": "tagval1"},
				metricType:         datapoint.Gauge,
				originalMetricType: "summary",
				t:                  time.Now(),
			},
			fn: ac.AddSummary,
		},
		{
			name: "AddHistogram()",
			want: &testEmitter{
				measurement:        "histogram_measurement",
				fields:             map[string]interface{}{"dim1": "dimval1"},
				tags:               map[string]string{"tag1": "tagval1"},
				metricType:         datapoint.Gauge,
				originalMetricType: "histogram",
				t:                  time.Now(),
			},
			fn: ac.AddHistogram,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ac.emit = &testEmitter{}
			tt.fn(tt.want.measurement, tt.want.fields, tt.want.tags, tt.want.t)
			if ac.emit.(*testEmitter).measurement != tt.want.measurement ||
				!reflect.DeepEqual(ac.emit.(*testEmitter).fields, tt.want.fields) ||
				!reflect.DeepEqual(ac.emit.(*testEmitter).tags, tt.want.tags) {
				t.Errorf("Accumulator_AddFields() = %v, want %v", ac.emit, tt.want)
			}
		})
	}
	t.Run("SetPrecision()", func(t *testing.T) {
		ac.emit = &testEmitter{}
		ac.SetPrecision(time.Second*1, time.Second*1)
	})
	t.Run("AddError()", func(t *testing.T) {
		ac.emit = &testEmitter{}
		err := fmt.Errorf("Test Error")
		ac.AddError(err)
		if ac.emit.(*testEmitter).err != err {
			t.Errorf("AddError() = %v, want %v", ac.emit.(*testEmitter).err, err)
		}
	})
}
