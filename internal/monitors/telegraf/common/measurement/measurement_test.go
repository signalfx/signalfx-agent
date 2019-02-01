package measurement

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
)

func TestMeasurement_RenameFieldWithTag(t *testing.T) {
	type fields struct {
		Measurement        string
		Fields             map[string]interface{}
		Tags               map[string]string
		MetricType         datapoint.MetricType
		OriginalMetricType string
		Timestamps         []time.Time
	}
	type args struct {
		tagName   string
		fieldName string
		replacer  *strings.Replacer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "replace field with tag",
			fields: fields{
				Measurement: "test_measurement",
				Fields: map[string]interface{}{
					"fieldKey": "fieldVal",
				},
				Tags: map[string]string{
					"tagKey": "tagVal",
				},
				MetricType:         datapoint.Gauge,
				OriginalMetricType: "unknown",
				Timestamps:         []time.Time{},
			},
			args: args{
				tagName:   "tagKey",
				fieldName: "fieldKey",
				replacer:  strings.NewReplacer(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.fields.Measurement,
				tt.fields.Fields,
				tt.fields.Tags,
				tt.fields.MetricType,
				tt.fields.OriginalMetricType,
				tt.fields.Timestamps...)
			tagVal := m.Tags[tt.args.tagName]
			fieldVal := m.Fields[tt.args.fieldName]
			fmt.Println(tagVal)
			fmt.Println(fieldVal)
			m.RenameFieldWithTag(tt.args.tagName, tt.args.fieldName, tt.args.replacer)
			fmt.Println(m)
			if m.Fields[tagVal] != fieldVal {
				t.Errorf("the new field's value (%v) does not equal the original field's value (%v)", m.Fields[tagVal], fieldVal)
			}
			if _, ok := m.Tags[tt.args.tagName]; ok {
				t.Error("original tag still exists in the measurement")
			}
			if _, ok := m.Fields[tt.args.fieldName]; ok {
				t.Error("original field still exists in the measurement")
			}
		})
	}
}
