package monitors

import (
	"fmt"
	"testing"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

func testMetadata(metricsExhaustive, sendAll bool) *Metadata {
	return &Metadata{
		MonitorType:       "test-monitor",
		IncludedMetrics:   utils.StringSet("cpu.idle", "cpu.min", "cpu.max", "mem.used"),
		Metrics:           utils.StringSet("cpu.idle", "cpu.min", "cpu.max", "mem.used", "mem.free", "mem.available"),
		MetricsExhaustive: metricsExhaustive,
		Groups:            utils.StringSet("cpu", "mem"),
		GroupMetricsMap: map[string][]string{
			// All cpu metrics are included.
			"cpu": {"cpu.idle", "cpu.min", "cpu.max"},
			// Only some mem metrics are included.
			"mem": {"mem.used", "mem.free", "mem.available"},
		},
		SendAll: sendAll,
	}
}

var exhaustiveMetadata = testMetadata(true, false)
var nonexhaustiveMetadata = testMetadata(false, false)
var sendAllMetadata = testMetadata(true, true)

func TestSendAll(t *testing.T) {
	if filter, err := newMetricsFilter(sendAllMetadata, nil); err != nil {
		t.Error(err)
	} else {
		// All included metrics should be sent.
		for metric := range sendAllMetadata.Metrics {
			t.Run(fmt.Sprintf("metric %s should send", metric), func(t *testing.T) {
				dp := &datapoint.Datapoint{
					Metric:     metric,
					MetricType: datapoint.Counter,
				}
				if !filter.shouldSend(dp) {
					t.Error()
				}
			})
		}
	}
}

func TestIncludedMetrics(t *testing.T) {
	if filter, err := newMetricsFilter(exhaustiveMetadata, nil); err != nil {
		t.Error(err)
	} else {
		// All included metrics should be sent.
		for metric := range exhaustiveMetadata.IncludedMetrics {
			t.Run(fmt.Sprintf("included metric %s should send", metric), func(t *testing.T) {
				dp := &datapoint.Datapoint{
					Metric:     metric,
					MetricType: datapoint.Counter,
				}
				if !filter.shouldSend(dp) {
					t.Error()
				}
			})
		}
	}
}

func TestAdditionalMetricNames(t *testing.T) {
	t.Run("user specifies already-included metric", func(t *testing.T) {
		if filter, err := newMetricsFilter(exhaustiveMetadata, []config.AdditionalMetric{
			{MetricName: "cpu.idle"}}); err != nil {
			t.Error(err)
		} else {
			if filter.additionalMetrics["cpu.idle"] {
				t.Error("cpu.idle should not have been in additional metrics because it is already included")
			}
		}
	})

	// Exhaustive
	if filter, err := newMetricsFilter(exhaustiveMetadata, []config.AdditionalMetric{
		{MetricName: "mem.used"}}); err != nil {
		t.Error(err)
	} else {
		for metric, shouldSend := range map[string]bool{
			"mem.used":      true,
			"mem.free":      false,
			"mem.available": false,
		} {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}
			sent := filter.shouldSend(dp)
			if sent && !shouldSend {
				t.Errorf("metric %s should not have sent", metric)
			}
			if !sent && shouldSend {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}

	// Non-exhaustive
	if filter, err := newMetricsFilter(nonexhaustiveMetadata, []config.AdditionalMetric{
		{MetricName: "dynamic-metric"},
		{MetricName: "some-*"}}); err != nil {
		t.Error(err)
	} else {
		for metric, shouldSend := range map[string]bool{
			"dynamic-metric":                  true,
			"some-globbed-metric":             true,
			"unconfigured-and-unknown-metric": false,
			"mem.used":                        true,
		} {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}
			sent := filter.shouldSend(dp)
			if sent && !shouldSend {
				t.Errorf("metric %s should not have sent", metric)
			}
			if !sent && shouldSend {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}

func TestGlobbedMetricNames(t *testing.T) {
	if filter, err := newMetricsFilter(exhaustiveMetadata, []config.AdditionalMetric{
		{MetricName: "mem.*"},
	}); err != nil {
		t.Error(err)
	} else {
		// All memory metrics should be sent.
		metrics := exhaustiveMetadata.GroupMetricsMap["mem"]
		if len(metrics) < 1 {
			t.Fatal("should be checking 1 or more metrics")
		}

		for _, metric := range metrics {
			dp := &datapoint.Datapoint{
				Metric:     metric,
				MetricType: datapoint.Counter,
			}
			if !filter.shouldSend(dp) {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}

func TestAdditionalMetricGroups(t *testing.T) {
	if filter, err := newMetricsFilter(exhaustiveMetadata, []config.AdditionalMetric{
		{Group: "mem"}}); err != nil {
		t.Error(err)
	} else {
		for _, metric := range exhaustiveMetadata.GroupMetricsMap["mem"] {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}

			if !filter.shouldSend(dp) {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}

func Test_newAdditionalMetricsFilter(t *testing.T) {
	type args struct {
		metadata          *Metadata
		additionalMetrics []config.AdditionalMetric
	}
	tests := []struct {
		name    string
		args    args
		wantNil bool
		wantErr bool
	}{
		// Exhaustive case errors.
		{"metricName is whitespace", args{exhaustiveMetadata, []config.AdditionalMetric{
			{MetricName: "    "},
		}}, true, true},
		{"groupName is whitespace", args{exhaustiveMetadata, []config.AdditionalMetric{
			{Group: "    "},
		}}, true, true},
		{"no group name or metric name", args{exhaustiveMetadata, []config.AdditionalMetric{{MetricName: "", Group: ""}}},
			true, true},
		{"group name and metric name", args{exhaustiveMetadata, []config.AdditionalMetric{{MetricName: "metric",
			Group: "group"}}}, true, true},
		{"unknown metric name", args{exhaustiveMetadata, []config.AdditionalMetric{{MetricName: "unknown-metric",
			Group: ""}}}, true, true},
		{"unknown group name", args{exhaustiveMetadata, []config.AdditionalMetric{{MetricName: "",
			Group: "unknown-group"}}}, true, true},
		{"metric glob doesn't match any metric", args{exhaustiveMetadata, []config.AdditionalMetric{
			{MetricName: "unknown-metric.*", Group: ""}}}, true, true},

		// Non-exhaustive cases.
		{"metricName is whitespace", args{nonexhaustiveMetadata, []config.AdditionalMetric{
			{MetricName: "    "},
		}}, true, true},
		{"groupName is whitespace", args{nonexhaustiveMetadata, []config.AdditionalMetric{
			{Group: "    "},
		}}, true, true},

		// Shouldn't error for non-exhaustive case.
		{"metric does not exist", args{nonexhaustiveMetadata, []config.AdditionalMetric{
			{MetricName: "unknown-metric"}}}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newMetricsFilter(tt.args.metadata, tt.args.additionalMetrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("newMetricsFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == nil) != tt.wantNil {
				t.Errorf("newMetricsFilter() = %v, want %v", got, tt.wantNil)
			}
		})
	}
}
