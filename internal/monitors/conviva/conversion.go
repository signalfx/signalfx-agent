package conviva

import (
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"strconv"
)

var metriclensMetrics = map[string][]string{
	"quality_metriclens": {"total_attempts","video_start_failures_percent","exits_before_video_start_percent","plays_percent","video_startup_time_sec","rebuffering_ratio_percent","average_bitrate_kbps","video_playback_failures_percent","ended_plays","connection_induced_rebuffering_ratio_percent","video_restart_time",},
	"audience_metriclens": {"concurrent_plays","plays", "ended_plays",},
}

func jsonResponseToDatapoints(jsonResponse map[string]interface{}) ([]*datapoint.Datapoint, error) {
	if jsonResponse["quality_metriclens"] != nil {
		return tableTypeResponseToDatapoints(jsonResponse, "quality_metriclens"), nil
	}
	return nil, nil
}

func tableTypeResponseToDatapoints(tableTypeResponse map[string]interface{}, metric string) []*datapoint.Datapoint {
	account     := tableTypeResponse["account"].(string)
	accountId   := tableTypeResponse["accountId"].(string)
	dimension   := tableTypeResponse["dimension"].(string)
	dimensionId := tableTypeResponse["dimensionId"].(string)
	filterNameById := tableTypeResponse["filterNameById"].(map[string]string)

	tables := tableTypeResponse[metric].(map[string]interface{})["tables"].(map[string]interface{})
	meta   := tableTypeResponse[metric].(map[string]interface{})["meta"].(map[string]interface{})
	status := strconv.FormatFloat(meta["status"].(float64), 'f', 0, 64)
	//if status != 0 {
	//}
	xvalues := tableTypeResponse[metric].(map[string]interface{})["xvalues"].([]interface{})

	var datapoints []*datapoint.Datapoint

	for filterId, table := range tables {
		filters_not_exist       := strconv.FormatBool(isFilterState(meta, "filters_not_exist", filterId))
		filters_incomplete_data := strconv.FormatBool(isFilterState(meta, "filters_incomplete_data", filterId))
		filters_warmup          := strconv.FormatBool(isFilterState(meta, "filters_warmup", filterId))

		for tableKey, tableValue := range table.(map[string]interface{}) {
			if tableKey == "rows" {
				for rowIndex, row := range tableValue.([]interface {}) {
					for metricIndex, metricValue := range row.([]interface {}) {
						dims := map[string]string{
							"metric":                      metric,
							"account":                     account,
							"account_id":                  accountId,
							"filter":                      filterNameById[filterId],
							"filter_id":                   filterId,
							"status":                      status,
							"filters_not_exist":           filters_not_exist,
							"filters_incomplete_data":     filters_incomplete_data,
							"filters_warmup":              filters_warmup,
							"metriclens_dimension":        dimension,
							"metriclens_dimension_id":     dimensionId,
							"metriclens_dimension_entity": xvalues[rowIndex].(string),
						}
						datapoints = append(datapoints, sfxclient.GaugeF(metriclensMetrics[metric][metricIndex], dims, metricValue.(float64)))
						//fmt.Printf("%v, %v = %v, %s, %t \n", dims, qualityMetricLensMetrics[metricIndex], metricValue.(float64), xvalues[rowIndex].(string))
					}
				}
			}
		}
	}
//	fmt.Printf("%v \n", datapoints)
	return datapoints
}

func isFilterState(meta map[string]interface{}, filterState string, filterId string) bool {
	if meta[filterState] != nil {
		for i := 0; i < len(meta[filterState].([]interface{})); i++ {
			if strconv.FormatFloat(meta[filterState].([]interface{})[i].(float64), 'f', 0, 64) == filterId {
				return true
			}
		}
	}
	return false
}
