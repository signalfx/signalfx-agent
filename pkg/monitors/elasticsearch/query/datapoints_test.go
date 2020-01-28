package query

import (
	"testing"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/stretchr/testify/assert"
)

// Tests datapoints collected from a bucket aggregation without
// sub metric aggregations
func TestDatapointsFromTerminalBucketAggregation(t *testing.T) {
	dps := collectDatapoints(
		HTTPResponse{
			Aggregations: map[string]*aggregationResponse{
				"host": {
					Buckets: map[interface{}]*bucketResponse{
						"helsniki": {
							Key:             "helsniki",
							DocCount:        newInt64(8800),
							SubAggregations: map[string]*aggregationResponse{},
						},
						"nairobi": {
							Key:             "nairobi",
							DocCount:        newInt64(8800),
							SubAggregations: map[string]*aggregationResponse{},
						},
					},
					SubAggregations: map[string]*aggregationResponse{},
					OtherValues:     map[string]interface{}{},
				},
			},
		}, map[string]*AggregationMeta{
			"host": {
				Type: "filters",
			},
		}, map[string]string{})

	assert.ElementsMatch(t, dps, []*datapoint.Datapoint{
		{
			Metric: "doc_count",
			Dimensions: map[string]string{
				"bucket_aggregation_name": "host",
				"host":                    "helsniki",
			},
			Value:      datapoint.NewIntValue(8800),
			MetricType: datapoint.Gauge,
		},
		{
			Metric: "doc_count",
			Dimensions: map[string]string{
				"bucket_aggregation_name": "host",
				"host":                    "nairobi",
			},
			Value:      datapoint.NewIntValue(8800),
			MetricType: datapoint.Gauge,
		},
	})
}

// Tests avg aggregation under a terms aggregation
func TestMetricAggregationWithTermsAggregation(t *testing.T) {
	dps := collectDatapoints(HTTPResponse{
		Aggregations: map[string]*aggregationResponse{
			"host": {
				Buckets: map[interface{}]*bucketResponse{
					"nairobi": {
						Key:      "nairobi",
						DocCount: newInt64(122),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								Value:           *newFloat64(48.41803278688525),
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
						},
					},
					"helsniki": {
						Key:      "helsniki",
						DocCount: newInt64(126),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								Value:           *newFloat64(49.357142857142854),
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
						},
					},
				},
				SubAggregations: map[string]*aggregationResponse{},
				OtherValues:     map[string]interface{}{},
			},
		},
	}, map[string]*AggregationMeta{
		"host": {
			Type: "terms",
		},
		"metric_agg_1": {
			Type: "avg",
		},
	}, map[string]string{})

	assert.ElementsMatch(t, dps, []*datapoint.Datapoint{
		{
			Metric: "avg",
			Dimensions: map[string]string{
				"metric_aggregation_name": "metric_agg_1",
				"host":                    "nairobi",
			},
			Value:      datapoint.NewFloatValue(48.41803278688525),
			MetricType: datapoint.Gauge,
		},
		{
			Metric: "avg",
			Dimensions: map[string]string{
				"metric_aggregation_name": "metric_agg_1",
				"host":                    "helsniki",
			},
			Value:      datapoint.NewFloatValue(49.357142857142854),
			MetricType: datapoint.Gauge,
		},
	})
}

// Tests datapoints from extended_stats aggregation within bucket aggregation
func TestExtendedStatsAggregationsFromFiltersAggregation(t *testing.T) {
	dps := collectDatapoints(HTTPResponse{
		Aggregations: map[string]*aggregationResponse{
			"host": {
				Buckets: map[interface{}]*bucketResponse{
					"nairobi": {
						Key:      "nairobi",
						DocCount: newInt64(5134),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues: map[string]interface{}{
									"count":          5134.0,
									"min":            0.0,
									"max":            100.0,
									"avg":            50.14530580444098,
									"sum":            257446.0,
									"sum_of_squares": 1.7184548E7,
									"variance":       832.6528246727477,
									"std_deviation":  28.855724296450223,
									"std_deviation_bounds": map[string]interface{}{
										"upper": 107.85675439734143,
										"lower": -7.566142788459466,
									},
								},
							},
						},
					},
					"madrid": {
						Key:      "madrid",
						DocCount: newInt64(5134),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues: map[string]interface{}{
									"count":          5134.0,
									"min":            0.0,
									"max":            100.0,
									"avg":            50.03486560186989,
									"sum":            256879.0,
									"sum_of_squares": 1.7288541E7,
									"variance":       863.9724891034797,
									"std_deviation":  29.39340893981982,
									"std_deviation_bounds": map[string]interface{}{
										"upper": 108.82168348150952,
										"lower": -8.751952277769753,
									},
								},
							},
						},
					},
				},
				SubAggregations: map[string]*aggregationResponse{},
				OtherValues:     map[string]interface{}{},
			},
		},
	}, map[string]*AggregationMeta{
		"host": {
			Type: "filters",
		},
		"metric_agg_1": {
			Type: "extended_stats",
		},
	}, map[string]string{})

	dims := map[string]map[string]string{
		"madrid": {
			"host":                    "madrid",
			"metric_aggregation_name": "metric_agg_1",
		},
		"nairobi": {
			"host":                    "nairobi",
			"metric_aggregation_name": "metric_agg_1",
		},
	}

	assert.ElementsMatch(t, dps, []*datapoint.Datapoint{
		{
			Metric:     "extended_stats.count",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(5134.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.min",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(0.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.max",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(100.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.avg",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(50.03486560186989),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(256879.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum_of_squares",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(1.7288541E7),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.variance",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(863.9724891034797),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(29.39340893981982),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.lower",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(-8.751952277769753),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.upper",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(108.82168348150952),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.count",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(5134.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.min",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(0.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.max",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(100.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.avg",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(50.14530580444098),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(257446.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum_of_squares",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(1.7184548E7),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.variance",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(832.6528246727477),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(28.855724296450223),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.lower",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(-7.566142788459466),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.upper",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(107.85675439734143),
			MetricType: datapoint.Gauge,
		},
	})
}

// Tests datapoints from percentiles aggregation within bucket aggregation
func TestPercentilesAggregationsFromFiltersAggregation(t *testing.T) {
	dps := collectDatapoints(HTTPResponse{
		Aggregations: map[string]*aggregationResponse{
			"host": {
				Buckets: map[interface{}]*bucketResponse{
					"nairobi": {
						Key:      "nairobi",
						DocCount: newInt64(5134),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								Values: map[string]interface{}{
									"50.0": 50.0,
									"75.0": 75.0,
									"99.0": 99.07999999999993,
								},
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
						},
					},
					"madrid": {
						Key:      "madrid",
						DocCount: newInt64(5134),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								Values: map[string]interface{}{
									"50.0": 50.294871794871796,
									"75.0": 75.98039215686275,
									"99.0": 100.0,
								},
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
						},
					},
				},
				SubAggregations: map[string]*aggregationResponse{},
				OtherValues:     map[string]interface{}{},
			},
		},
	}, map[string]*AggregationMeta{
		"host": {
			Type: "filters",
		},
		"metric_agg_1": {
			Type: "percentiles",
		},
	}, map[string]string{})

	dims := map[string]map[string]string{
		"madrid": {
			"host":                    "madrid",
			"metric_aggregation_name": "metric_agg_1",
		},
		"nairobi": {
			"host":                    "nairobi",
			"metric_aggregation_name": "metric_agg_1",
		},
	}

	assert.ElementsMatch(t, dps, []*datapoint.Datapoint{
		{
			Metric:     "percentiles.p50",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(50.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p75",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(75.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p99",
			Dimensions: dims["nairobi"],
			Value:      datapoint.NewFloatValue(99.07999999999993),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p50",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(50.294871794871796),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p75",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(75.98039215686275),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p99",
			Dimensions: dims["madrid"],
			Value:      datapoint.NewFloatValue(100.0),
			MetricType: datapoint.Gauge,
		},
	})
}

// Tests multiple metric aggregation in a bucket aggregation
func TestMultipleMetricAggregationWithTermsAggregation(t *testing.T) {
	dps := collectDatapoints(HTTPResponse{
		Aggregations: map[string]*aggregationResponse{
			"host": {
				Buckets: map[interface{}]*bucketResponse{
					"nairobi": {
						Key:      "nairobi",
						DocCount: newInt64(5134),
						SubAggregations: map[string]*aggregationResponse{
							"metric_agg_1": {
								Values: map[string]interface{}{
									"50.0": 50.0,
									"75.0": 75.0,
									"99.0": 99.07999999999993,
								},
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
							"metric_agg_2": {
								Value:           *newFloat64(48.41803278688525),
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues:     map[string]interface{}{},
							},
							"metric_agg_3": {
								SubAggregations: map[string]*aggregationResponse{},
								OtherValues: map[string]interface{}{
									"count":          5134.0,
									"min":            0.0,
									"max":            100.0,
									"avg":            50.14530580444098,
									"sum":            257446.0,
									"sum_of_squares": 1.7184548E7,
									"variance":       832.6528246727477,
									"std_deviation":  28.855724296450223,
									"std_deviation_bounds": map[string]interface{}{
										"upper": 107.85675439734143,
										"lower": -7.566142788459466,
									},
								},
							},
						},
					},
				},
				SubAggregations: map[string]*aggregationResponse{},
				OtherValues:     map[string]interface{}{},
			},
		},
	}, map[string]*AggregationMeta{
		"host": {
			Type: "filters",
		},
		"metric_agg_1": {
			Type: "percentiles",
		},
		"metric_agg_2": {
			Type: "avg",
		},
		"metric_agg_3": {
			Type: "extended_stats",
		},
	}, map[string]string{})

	dims := map[string]map[string]string{
		"metric_agg_1": {
			"host":                    "nairobi",
			"metric_aggregation_name": "metric_agg_1",
		},
		"metric_agg_2": {
			"host":                    "nairobi",
			"metric_aggregation_name": "metric_agg_2",
		},
		"metric_agg_3": {
			"host":                    "nairobi",
			"metric_aggregation_name": "metric_agg_3",
		},
	}

	assert.ElementsMatch(t, dps, []*datapoint.Datapoint{
		{
			Metric:     "percentiles.p50",
			Dimensions: dims["metric_agg_1"],
			Value:      datapoint.NewFloatValue(50.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p75",
			Dimensions: dims["metric_agg_1"],
			Value:      datapoint.NewFloatValue(75.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "percentiles.p99",
			Dimensions: dims["metric_agg_1"],
			Value:      datapoint.NewFloatValue(99.07999999999993),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "avg",
			Dimensions: dims["metric_agg_2"],
			Value:      datapoint.NewFloatValue(48.41803278688525),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.count",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(5134.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.min",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(0.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.max",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(100.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.avg",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(50.14530580444098),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(257446.0),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.sum_of_squares",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(1.7184548E7),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.variance",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(832.6528246727477),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(28.855724296450223),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.lower",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(-7.566142788459466),
			MetricType: datapoint.Gauge,
		},
		{
			Metric:     "extended_stats.std_deviation_bounds.upper",
			Dimensions: dims["metric_agg_3"],
			Value:      datapoint.NewFloatValue(107.85675439734143),
			MetricType: datapoint.Gauge,
		},
	})
}
