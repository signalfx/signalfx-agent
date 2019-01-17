package perfcounter

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/monitors/telegraf/common/measurement"
	"github.com/signalfx/signalfx-agent/internal/neotest"
)

func TestLogicalDiskMeasurement(t *testing.T) {
	var monitorType = "signalfx-system-utilization"
	var Measurement = "win_logical_disk"
	var percentFreeSpace = float32(80)
	var utilization = float64(100 - percentFreeSpace)
	var freeMegabytes = float32(1024)
	var usedBytes = ((float64(freeMegabytes) * megabytesToBytes) * 100 / (100 - utilization)) - (float64(freeMegabytes) * megabytesToBytes)
	type args struct {
		ms          []*measurement.Measurement
		monitorType string
		output      *neotest.TestOutput
	}
	tests := []struct {
		name       string
		args       args
		perf       PerfCounter
		want       []*datapoint.Datapoint
		wantErrors []error
	}{
		{
			name: "disk.utilization",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Free_Space": percentFreeSpace,
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: LogicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("disk.utilization", map[string]string{"plugin": monitorType, "plugin_instance": "C:"}, datapoint.NewFloatValue(utilization), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "disk.utilization - bad value",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Free_Space": float64(percentFreeSpace),
						},
						Tags: map[string]string{
							"instance": "C:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       LogicalDisk(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("error parsing value '%v' for 'Percent_Free_Space' field in logical disk measurement '%s'", float64(percentFreeSpace), Measurement)},
		},
		{
			name: "df_complex",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Free_Space": percentFreeSpace,
							"Free_Megabytes":     freeMegabytes,
						},
						Tags: map[string]string{
							"instance": "D:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: LogicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("disk.utilization", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewFloatValue(utilization), datapoint.Gauge, time.Time{}),
				datapoint.New("df_complex.free", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewFloatValue(float64(freeMegabytes)*megabytesToBytes), datapoint.Gauge, time.Time{}),
				datapoint.New("df_complex.used", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewFloatValue(usedBytes), datapoint.Gauge, time.Time{}),
			},
			wantErrors: nil,
		},
		{
			name: "df_complex - bad value",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Free_Space": percentFreeSpace,
							"Free_Megabytes":     float64(freeMegabytes),
						},
						Tags: map[string]string{
							"instance": "D:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: LogicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("disk.utilization", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewFloatValue(utilization), datapoint.Gauge, time.Time{}),
			},
			wantErrors: []error{fmt.Errorf("error parsing value '%v' for 'Free_Megabytes' field in logical disk measurement '%s'", float64(freeMegabytes), Measurement)},
		},
		{
			name: "missing Percent_Free_Space field",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Free_Megabytes": freeMegabytes,
						},
						Tags: map[string]string{
							"instance": "D:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf: LogicalDisk(),
			want: []*datapoint.Datapoint{
				datapoint.New("df_complex.free", map[string]string{"plugin": monitorType, "plugin_instance": "D:"}, datapoint.NewFloatValue(float64(freeMegabytes)*megabytesToBytes), datapoint.Gauge, time.Time{}),
			},
			wantErrors: []error{fmt.Errorf("No 'Percent_Free_Space' field on logical disk measurement '%s'", Measurement)},
		},
		{
			name: "no fields",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields:      map[string]interface{}{},
						Tags:        map[string]string{},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       LogicalDisk(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no fields on logical disk measurement '%s'", Measurement)},
		},
		{
			name: "no instance",
			args: args{
				ms: []*measurement.Measurement{
					{
						Measurement: Measurement,
						Fields: map[string]interface{}{
							"Percent_Free_Space": percentFreeSpace,
							"Free_Megabytes":     freeMegabytes,
						},
						Tags: map[string]string{
							"reinstance": "D:",
						},
					},
				},
				monitorType: monitorType,
				output:      neotest.NewTestOutput(),
			},
			perf:       LogicalDisk(),
			want:       nil,
			wantErrors: []error{fmt.Errorf("no instance tag defined in tags 'map[reinstance:D:]' for measurement '%s'", Measurement)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, ms := range tt.args.ms {
				if gotErrors := tt.perf.ProcessMeasurement(ms, tt.args.monitorType, tt.args.output.SendDatapoint); !reflect.DeepEqual(gotErrors, tt.wantErrors) {
					t.Errorf("ProcessMeasurement() = %v, want %v", gotErrors, tt.wantErrors)
					continue
				}
			}
			if dps := tt.args.output.FlushDatapoints(); !reflect.DeepEqual(tt.want, dps) {
				t.Errorf("ProcessMeasurement() = %v, want %v", dps, tt.want)
			}
		})
	}
}
