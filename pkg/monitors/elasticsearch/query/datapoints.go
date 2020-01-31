package query

import (
	"fmt"
	"strconv"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	log "github.com/sirupsen/logrus"
)

// Struct keeps track of info required at a level of recursion
// An instance of this struct can be thought of as a datapoint
// collector for a particular aggregation
type dpCollector struct {
	aggName       string
	aggRes        aggregationResponse
	aggsMeta      map[string]*AggregationMeta
	sfxDimensions map[string]string
}

// Returns aggregation type
func (dpC *dpCollector) getType() string {
	return dpC.aggsMeta[dpC.aggName].Type
}

// Walks through the response, collecting dimensions and datapoints depending on the
// type of aggregation at each recursive level
func collectDatapoints(resBody HTTPResponse, aggsMeta map[string]*AggregationMeta, sfxDimensions map[string]string) []*datapoint.Datapoint {
	out := make([]*datapoint.Datapoint, 0)
	aggsResult := resBody.Aggregations

	for k, v := range aggsResult {
		// each aggregation at the highest level starts with an empty set of dimensions
		out = append(out, (&dpCollector{
			aggName:       k,
			aggRes:        *v,
			aggsMeta:      aggsMeta,
			sfxDimensions: sfxDimensions,
		}).recursivelyCollectDatapoints()...)
	}

	return out
}

func (dpC *dpCollector) recursivelyCollectDatapoints() []*datapoint.Datapoint {
	sfxDatapoints := make([]*datapoint.Datapoint, 0)

	// The absence of "doc_count" and "buckets" field is a good indicator that
	// the aggregation is a metric aggregation
	if isMetricAggregation(&dpC.aggRes) {
		return dpC.collectDatapointsFromMetricAggregation()
	}

	// Recursively collect all datapoints from buckets at this level
	for _, b := range dpC.aggRes.Buckets {
		key, ok := b.Key.(string)

		if !ok {
			log.WithFields(
				log.Fields{
					"aggregation_name": dpC.aggName,
					"aggregation_type": dpC.getType(),
				}).Warn("Found non string key for bucket. Skipping current aggregation and sub aggregations")
			break
		}

		// Pick the current bucket's key as a dimension before recursing down to the next level
		sfxDimensionsForBucket := utils.CloneStringMap(dpC.sfxDimensions)
		sfxDimensionsForBucket[dpC.aggName] = key

		// Send document count as metrics when there are no metric aggregations specified
		// under a bucket aggregation and there aren't sub aggregations as well
		if isTerminalBucket(b) {
			sfxDatapoints = append(sfxDatapoints, collectDocCountFromTerminalBucket(b, dpC.aggName, sfxDimensionsForBucket)...)
			continue
		}

		for k, v := range b.SubAggregations {
			sfxDatapoints = append(sfxDatapoints, (&dpCollector{
				aggName:       k,
				aggRes:        *v,
				aggsMeta:      dpC.aggsMeta,
				sfxDimensions: sfxDimensionsForBucket,
			}).recursivelyCollectDatapoints()...)
		}
	}

	// Recursively collect datapoints from sub aggregations
	for k, v := range dpC.aggRes.SubAggregations {
		sfxDatapoints = append(sfxDatapoints, (&dpCollector{
			aggName:       k,
			aggRes:        *v,
			aggsMeta:      dpC.aggsMeta,
			sfxDimensions: dpC.sfxDimensions,
		}).recursivelyCollectDatapoints()...)
	}

	return sfxDatapoints
}

// Collects "doc_count" from a bucket as a SFx datapoint if a bucket aggregation
// does not have sub metric aggregations
func collectDocCountFromTerminalBucket(bucket *bucketResponse, aggName string, dims map[string]string) []*datapoint.Datapoint {
	dimsForBucket := utils.CloneStringMap(dims)
	dimsForBucket["bucket_aggregation_name"] = aggName

	out, ok := collectDatapoint("doc_count", *bucket.DocCount, dimsForBucket)

	if !ok {
		return []*datapoint.Datapoint{}
	}

	return []*datapoint.Datapoint{out}
}

// Collects datapoints from supported metric aggregations
func (dpC *dpCollector) collectDatapointsFromMetricAggregation() []*datapoint.Datapoint {

	out := make([]*datapoint.Datapoint, 0)

	// Add metric aggregation name as a dimension
	sfxDimensionsForMetric := utils.CloneStringMap(dpC.sfxDimensions)
	sfxDimensionsForMetric["metric_aggregation_name"] = dpC.aggName

	aggType := dpC.getType()
	switch aggType {
	case "stats":
		fallthrough
	case "extended_stats":
		out = append(out, getDatapointsFromStats(aggType, &dpC.aggRes, sfxDimensionsForMetric)...)
	case "percentiles":
		out = append(out, getDatapointsFromPercentiles(&dpC.aggRes, sfxDimensionsForMetric)...)
	default:
		metricName := aggType
		dp, ok := collectDatapoint(metricName, dpC.aggRes.Value, sfxDimensionsForMetric)

		if !ok {
			log.WithFields(log.Fields{"aggregation_type": aggType,
				"aggregation_name": dpC.aggName}).Warnf("Invalid value found: %v", dpC.aggRes.Value)
			return out
		}

		out = append(out, dp)
	}

	return out
}

// Collect datapoints from "stats" or "extended_stats" metric aggregation
// Extended stats aggregations look like:
// {
//		"count" : 36370,
//		"min" : 0.0,
//		"max" : 100.0,
//		"avg" : 49.98350288699478,
//		"sum" : 1817900.0,
//		"sum_of_squares" : 1.21849642E8,
//		"variance" : 851.9282953459498,
//		"std_deviation" : 29.187810732323687,
//		"std_deviation_bounds" : {
//			"upper" : 108.35912435164215,
//			"lower" : -8.392118577652596
//  	}
// }
// Metric names from this integration will look like "extended_stats.count",
// "extended_stats.min", "extended_stats.std_deviation_bounds.lower" and so on
func getDatapointsFromStats(aggType string, aggRes *aggregationResponse, dims map[string]string) []*datapoint.Datapoint {
	aggName := dims["metric_aggregation_name"]
	out := make([]*datapoint.Datapoint, 0)

	for k, v := range aggRes.OtherValues {
		switch k {
		case "std_deviation_bounds":
			m, ok := v.(map[string]interface{})

			if !ok {
				log.WithFields(log.Fields{"extended_stat": k,
					"aggregation_name": aggName}).Warnf("Invalid value found for stat: %v", v)
				continue
			}

			for bk, bv := range m {
				metricName := fmt.Sprintf("%s.%s.%s", aggType, k, bk)
				dp, ok := collectDatapoint(metricName, bv, dims)

				if !ok {
					log.WithFields(log.Fields{"stat": k,
						"aggregation_name": aggName}).Warnf("Invalid value found for stat: %v", bv)
					continue
				}

				out = append(out, dp)
			}
		default:
			metricName := fmt.Sprintf("%s.%s", aggType, k)
			dp, ok := collectDatapoint(metricName, v, dims)

			if !ok {
				log.WithFields(log.Fields{"stat": k,
					"aggregation_name": aggName}).Warnf("Invalid value found for stat: %v", v)
				continue
			}

			out = append(out, dp)
		}
	}

	return out
}

// Collect datapoint from "percentiles" metric aggregation
func getDatapointsFromPercentiles(aggRes *aggregationResponse, dims map[string]string) []*datapoint.Datapoint {
	aggName := dims["metric_aggregation_name"]
	out := make([]*datapoint.Datapoint, 0)

	// Values are always expected to be a map between the percentile and the
	// actual value itself of the metric
	values, ok := aggRes.Values.(map[string]interface{})

	if !ok {
		log.WithFields(log.Fields{"aggregation_name": aggName}).Warnf("No valid values found in percentiles aggregation")
	}

	// Metric name is constituted of the aggregation type "percentiles" and the actual percentile
	// Metric names from this aggregation will look like "percentiles.p99", "percentiles.p50" and
	// the aggregation name used to compute the metric will be a sent in as "metric_aggregation_name"
	// dimension on the datapoint
	for k, v := range values {
		p, err := strconv.ParseFloat(k, 64)

		if err != nil {
			log.WithFields(log.Fields{"aggregation_name": aggName}).Warnf("Invalid percentile found: %s", k)
			continue
		}

		// Remove trailing zeros
		metricName := fmt.Sprintf("%s.p%s", "percentiles", strconv.FormatFloat(p, 'f', -1, 64))
		dp, ok := collectDatapoint(metricName, v, dims)

		if !ok {
			log.WithFields(log.Fields{"percentile": k,
				"aggregation_name": aggName}).Warnf("Invalid value found for percentile: %v", v)
			continue
		}

		out = append(out, dp)
	}

	return out
}

// Returns true if aggregation is a metric aggregation
func isMetricAggregation(aggRes *aggregationResponse) bool {
	return aggRes.DocCount == nil && len(aggRes.Buckets) == 0
}

// Returns true if bucket aggregation is at the deepest level without
// sub metric aggregations
func isTerminalBucket(b *bucketResponse) bool {
	return len(b.SubAggregations) == 0 && b.DocCount != nil
}

// Collects a single datapoint from an interface, returns false if no datapoint can be derived
func collectDatapoint(metricName string, value interface{}, dims map[string]string) (*datapoint.Datapoint, bool) {

	out := datapoint.Datapoint{
		Metric:     fmt.Sprintf("%s.%s", "elasticsearch_query", metricName),
		Dimensions: dims,
		MetricType: datapoint.Gauge,
	}

	switch v := value.(type) {
	case float64:
		out.Value = datapoint.NewFloatValue(v)
	case int64:
		out.Value = datapoint.NewIntValue(v)
	case *float64:
		out.Value = datapoint.NewFloatValue(*v)
	case *int64:
		out.Value = datapoint.NewIntValue(*v)
	default:
		return nil, false
	}

	return &out, true
}
