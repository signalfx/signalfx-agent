package monitors

import (
	"fmt"
	"testing"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

func testMetadata(metricsExhaustive, sendAll bool) *Metadata {
	return &Metadata{
		MonitorType:     "test-monitor",
		IncludedMetrics: utils.StringSet("cpu.idle", "cpu.min", "cpu.max", "mem.used"),
		Metrics: map[string]MetricInfo{
			"cpu.idle":      {datapoint.Gauge},
			"cpu.min":       {datapoint.Gauge},
			"cpu.max":       {datapoint.Gauge},
			"mem.used":      {datapoint.Counter},
			"mem.free":      {datapoint.Counter},
			"mem.available": {datapoint.Counter}},
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
	if filter, err := newMetricsFilter(sendAllMetadata, nil, nil); err != nil {
		t.Error(err)
	} else {
		// All included metrics should be sent.
		for metric := range sendAllMetadata.Metrics {
			t.Run(fmt.Sprintf("metric %s should send", metric), func(t *testing.T) {
				dp := &datapoint.Datapoint{
					Metric:     metric,
					MetricType: datapoint.Counter,
				}
				if !filter.Matches(dp) {
					t.Error()
				}
			})
		}
	}
}

func TestIncludedMetrics(t *testing.T) {
	if filter, err := newMetricsFilter(exhaustiveMetadata, nil, nil); err != nil {
		t.Error(err)
	} else {
		// All included metrics should be sent.
		for metric := range exhaustiveMetadata.IncludedMetrics {
			t.Run(fmt.Sprintf("included metric %s should send", metric), func(t *testing.T) {
				dp := &datapoint.Datapoint{
					Metric:     metric,
					MetricType: datapoint.Counter,
				}
				if !filter.Matches(dp) {
					t.Error()
				}
			})
		}
	}
}

func TestExtraMetrics(t *testing.T) {
	t.Run("user specifies already-included metric", func(t *testing.T) {
		if filter, err := newMetricsFilter(exhaustiveMetadata, []string{"cpu.idle"}, nil); err != nil {
			t.Error(err)
		} else {
			if filter.extraMetrics["cpu.idle"] {
				t.Error("cpu.idle should not have been in additional metrics because it is already included")
			}
		}
	})

	// Exhaustive
	if filter, err := newMetricsFilter(exhaustiveMetadata, []string{"mem.used"}, nil); err != nil {
		t.Error(err)
	} else {
		for metric, shouldSend := range map[string]bool{
			"mem.used":      true,
			"mem.free":      false,
			"mem.available": false,
		} {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}
			sent := filter.Matches(dp)
			if sent && !shouldSend {
				t.Errorf("metric %s should not have sent", metric)
			}
			if !sent && shouldSend {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}

	// Non-exhaustive
	if filter, err := newMetricsFilter(nonexhaustiveMetadata, []string{"dynamic-metric", "some-*"}, nil); err != nil {
		t.Error(err)
	} else {
		for metric, shouldSend := range map[string]bool{
			"dynamic-metric":                  true,
			"some-globbed-metric":             true,
			"unconfigured-and-unknown-metric": false,
			"mem.used":                        true,
		} {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}
			sent := filter.Matches(dp)
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
	if filter, err := newMetricsFilter(exhaustiveMetadata, []string{"mem.*"}, nil); err != nil {
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
			if !filter.Matches(dp) {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}

func TestExtraMetricGroups(t *testing.T) {
	if filter, err := newMetricsFilter(exhaustiveMetadata, nil, []string{"mem"}); err != nil {
		t.Error(err)
	} else {
		for _, metric := range exhaustiveMetadata.GroupMetricsMap["mem"] {
			dp := &datapoint.Datapoint{Metric: metric, MetricType: datapoint.Counter}

			if !filter.Matches(dp) {
				t.Errorf("metric %s should have been sent", metric)
			}
		}
	}
}

func Test_newExtraMetricsFilter(t *testing.T) {
	type args struct {
		metadata     *Metadata
		extraMetrics []string
		extraGroups  []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Error cases.
		{"unknown group name", args{exhaustiveMetadata, nil, []string{"unknown-group"}}, true},
		{"metricName is whitespace", args{exhaustiveMetadata, []string{"    "}, nil}, true},
		{"groupName is whitespace", args{exhaustiveMetadata, nil, []string{"    "}}, true},
		{"metricName is whitespace", args{nonexhaustiveMetadata, []string{"    "}, nil}, true},
		{"groupName is whitespace", args{nonexhaustiveMetadata, nil, []string{"    "}}, true},

		// Successful cases.
		{"no group name or metric name", args{exhaustiveMetadata, nil, nil}, false},
		{"valid group name and unknown metric name", args{exhaustiveMetadata, []string{"unknown-metric"},
			[]string{"cpu"}}, false},
		{"unknown metric name", args{exhaustiveMetadata, []string{"unknown-metric"}, nil}, false},
		{"metric glob doesn't match any metric", args{exhaustiveMetadata, []string{"unknown-metric.*"}, nil}, false},
		{"metric does not exist", args{nonexhaustiveMetadata, []string{"unknown-metric"}, nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newMetricsFilter(tt.args.metadata, tt.args.extraMetrics, tt.args.extraGroups)
			if (err != nil) != tt.wantErr {
				t.Errorf("newMetricsFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
