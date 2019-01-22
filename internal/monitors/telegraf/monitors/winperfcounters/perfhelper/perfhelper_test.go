package perfhelper

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/neotest"
)

func TestProcessMeasurements(t *testing.T) {
	type args struct {
		measurements       []*measurement.Measurement
		mappings           map[string]*MetricMapper
		sendDatapoint      func(dp *datapoint.Datapoint)
		defaultMonitorType string
		defaultInstanceKey string
	}
	tests := []struct {
		name       string
		args       args
		output     *neotest.TestOutput
		want       []*datapoint.Datapoint
		wantErrors []error
	}{
		{
			name: "process measurement without overrides",
			args: args{
				measurements: []*measurement.Measurement{
					{
						Measurement: "win_physical_disk",
						Fields: map[string]interface{}{
							"Disk_Reads_persec": 80,
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				defaultMonitorType: "disk_ops",
				defaultInstanceKey: "instance",
			},
			output: neotest.NewTestOutput(),
			want: []*datapoint.Datapoint{
				datapoint.New("win_physical_disk.disk_reads_persec", map[string]string{"plugin": "disk_ops", "instance": "c_"}, datapoint.NewIntValue(80), datapoint.Gauge, time.Time{}),
			},
		},
		{
			name: "process measurement with overrides",
			args: args{
				measurements: []*measurement.Measurement{
					{
						Measurement: "win_physical_disk",
						Fields: map[string]interface{}{
							"Disk_Reads_persec": 80,
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				mappings: map[string]*MetricMapper{
					"win_physical_disk.disk_reads_persec": {
						Name:     "alternate_metric_name",
						Type:     datapoint.Counter,
						Monitor:  "alternate_monitor_name",
						Instance: "alternate_instance",
					},
				},
				defaultMonitorType: "disk_ops",
				defaultInstanceKey: "instance",
			},
			output: neotest.NewTestOutput(),
			want: []*datapoint.Datapoint{
				datapoint.New("alternate_metric_name", map[string]string{"plugin": "alternate_monitor_name", "alternate_instance": "c_"}, datapoint.NewIntValue(80), datapoint.Counter, time.Time{}),
			},
		},
		{
			name: "process measurement with no fields",
			args: args{
				measurements: []*measurement.Measurement{
					{
						Measurement: "win_physical_disk",
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				defaultMonitorType: "disk_ops",
				defaultInstanceKey: "instance",
			},
			output:     neotest.NewTestOutput(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no fields on measurement 'win_physical_disk'")},
		},
		{
			name: "process measurement with no instance",
			args: args{
				measurements: []*measurement.Measurement{
					{
						Measurement: "win_physical_disk",
						Fields: map[string]interface{}{
							"Disk_Reads_persec": 80,
						},
						Tags: map[string]string{},
					},
				},
				defaultMonitorType: "disk_ops",
				defaultInstanceKey: "instance",
			},
			output:     neotest.NewTestOutput(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no instance tag defined in tags 'map[]' for measurement 'win_physical_disk'")},
		},
		{
			name: "process measurement with invalid value",
			args: args{
				measurements: []*measurement.Measurement{
					{
						Measurement: "win_physical_disk",
						Fields: map[string]interface{}{
							"Disk_Reads_persec": "Bad Value",
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				defaultMonitorType: "disk_ops",
				defaultInstanceKey: "instance",
			},
			output:     neotest.NewTestOutput(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("unknown metric value type string")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotErrors := ProcessMeasurements(tt.args.measurements, tt.args.mappings, tt.output.SendDatapoint, tt.args.defaultMonitorType, tt.args.defaultInstanceKey); !reflect.DeepEqual(gotErrors, tt.wantErrors) {
				t.Errorf("ProcessMeasurement() = %v, want %v", gotErrors, tt.wantErrors)
			}
			if dps := tt.output.FlushDatapoints(); !reflect.DeepEqual(tt.want, dps) {
				t.Errorf("ProcessMeasurement() = %v, want %v", dps, tt.want)
			}
		})
	}
}
