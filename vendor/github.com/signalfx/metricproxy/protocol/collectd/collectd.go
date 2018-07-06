package collectd

import (
	"strings"
	"time"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/event"
	"github.com/signalfx/metricproxy/protocol/collectd/format"
)

var dsTypeToMetricType = map[string]datapoint.MetricType{
	"gauge":    datapoint.Gauge,
	"derive":   datapoint.Counter,
	"counter":  datapoint.Counter,
	"absolute": datapoint.Count,
}

// JSONWriteFormat is an alias
type JSONWriteFormat collectdformat.JSONWriteFormat

// JSONWriteBody is an alias
type JSONWriteBody collectdformat.JSONWriteBody

func metricTypeFromDsType(dstype *string) datapoint.MetricType {
	if dstype == nil {
		return datapoint.Gauge
	}

	v, ok := dsTypeToMetricType[*dstype]
	if ok {
		return v
	}
	return datapoint.Gauge
}

func isNilOrEmpty(str *string) bool {
	return str == nil || *str == ""
}

// NewDatapoint creates a new datapoint from collectd's write_http endpoint JSON format
// defaultDimensions are added to the datapoint created, but will be overridden by any dimension
// values in the JSON
// Dimensions are pulled out of type_instance, plugin_instance and host in that order of precedence
func NewDatapoint(point *JSONWriteFormat, index uint, defaultDimensions map[string]string) *datapoint.Datapoint {
	dstype, val, dsname := point.Dstypes[index], point.Values[index], point.Dsnames[index]
	// if you add another  dimension that we read from the json update this number
	const MaxCollectDDims = 6
	dimensions := make(map[string]string, len(defaultDimensions)+MaxCollectDDims)
	for k, v := range defaultDimensions {
		dimensions[k] = v
	}

	metricType := metricTypeFromDsType(dstype)
	metricName, usedDsName := getReasonableMetricName(point, index, dimensions)

	addIfNotNullOrEmpty(dimensions, "plugin", true, point.Plugin)

	parseDimensionsOut(dimensions, point.PluginInstance, point.Host)

	addIfNotNullOrEmpty(dimensions, "dsname", !usedDsName, dsname)

	timestamp := time.Unix(0, int64(float64(time.Second)**point.Time))
	var value datapoint.Value
	if *val == float64(int64(*val)) {
		value = datapoint.NewIntValue(int64(*val))
	} else {
		value = datapoint.NewFloatValue(*val)
	}
	return datapoint.New(metricName, dimensions, value, metricType, timestamp)
}

func addIfNotNullOrEmpty(dimensions map[string]string, key string, cond bool, val *string) {
	if cond && val != nil && *val != "" {
		dimensions[key] = *val
	}
}

func parseDimensionsOut(dimensions map[string]string, pluginInstance *string, host *string) {
	parseNameForDimensions(dimensions, "plugin_instance", pluginInstance)
	parseNameForDimensions(dimensions, "host", host)
}

// GetDimensionsFromName tries to pull out dimensions out of name in the format name[k=v,f=x]-morename
// would return name-morename and extract dimensions (k,v) and (f,x)
// if we encounter something we don't expect use original
// this is a bit complicated to avoid allocations, string.split allocates, while slices
// inside same function, do not.
func GetDimensionsFromName(val *string) (instanceName string, toAddDims map[string]string) {
	instanceName = *val
	index := strings.Index(*val, "[")
	if index > -1 {
		left := (*val)[:index]
		rest := (*val)[index+1:]
		index = strings.Index(rest, "]")
		if index > -1 {
			working := make(map[string]string)
			dimensions := rest[:index]
			rest = rest[index+1:]
			cindex := strings.Index(dimensions, ",")
			prev := 0
			for {
				if cindex < prev {
					cindex = len(dimensions)
				}
				piece := dimensions[prev:cindex]
				tindex := strings.Index(piece, "=")
				if tindex == -1 || strings.Index(piece[tindex+1:], "=") > -1 {
					return
				}
				working[piece[:tindex]] = piece[tindex+1:]
				if cindex == len(dimensions) {
					break
				}
				prev = cindex + 1
				cindex = strings.Index(dimensions[prev:], ",") + prev
			}
			toAddDims = working
			instanceName = left + rest
		}
	}
	return
}

func parseNameForDimensions(dimensions map[string]string, key string, val *string) {
	instanceName, toAddDims := GetDimensionsFromName(val)

	for k, v := range toAddDims {
		if _, exists := dimensions[k]; !exists {
			addIfNotNullOrEmpty(dimensions, k, true, &v)
		}
	}
	addIfNotNullOrEmpty(dimensions, key, true, &instanceName)
}

func pointTypeInstance(point *JSONWriteFormat, dimensions map[string]string, parts []byte) []byte {
	if !isNilOrEmpty(point.TypeInstance) {
		instanceName, toAddDims := GetDimensionsFromName(point.TypeInstance)
		if instanceName != "" {
			if len(parts) > 0 {
				parts = append(parts, '.')
			}
			parts = append(parts, instanceName...)
		}
		for k, v := range toAddDims {
			if _, exists := dimensions[k]; !exists {
				addIfNotNullOrEmpty(dimensions, k, true, &v)
			}
		}
	}
	return parts
}

// getReasonableMetricName creates metrics names by joining them (if non empty) type.typeinstance
// if there are more than one dsname append .dsname for the particular uint. if there's only one it
// becomes a dimension
func getReasonableMetricName(point *JSONWriteFormat, index uint, dimensions map[string]string) (string, bool) {
	usedDsName := false
	parts := make([]byte, 0, len(*point.TypeS)+len(*point.TypeInstance))
	if !isNilOrEmpty(point.TypeS) {
		parts = append(parts, *point.TypeS...)
	}
	parts = pointTypeInstance(point, dimensions, parts)
	if point.Dsnames != nil && !isNilOrEmpty(point.Dsnames[index]) && len(point.Dsnames) > 1 {
		if len(parts) > 0 {
			parts = append(parts, '.')
		}
		parts = append(parts, *point.Dsnames[index]...)
		usedDsName = true
	}
	return string(parts), usedDsName
}

// NewEvent creates a new event from collectd's write_http endpoint JSON format
// defaultDimensions are added to the event created, but will be overridden by any dimension
// values in the JSON
func NewEvent(e *JSONWriteFormat, defaultDimensions map[string]string) *event.Event {
	// if you add another  dimension that we read from the json update this number
	const MaxCollectDDims = 6
	dimensions := make(map[string]string, len(defaultDimensions)+MaxCollectDDims)
	for k, v := range defaultDimensions {
		dimensions[k] = v
	}
	// events don't have a dsname
	eventType, _ := getReasonableMetricName(e, 0, dimensions)
	addIfNotNullOrEmpty(dimensions, "plugin", true, e.Plugin)
	parseDimensionsOut(dimensions, e.PluginInstance, e.Host)

	properties := make(map[string]interface{}, len(e.Meta)+2)
	for k, v := range e.Meta {
		properties[k] = v
	}
	_, exists := e.Meta["severity"]
	if !exists && e.Severity != nil {
		properties["severity"] = *e.Severity
	}
	_, exists = e.Meta["message"]
	if !exists && e.Message != nil {
		properties["message"] = *e.Message
	}

	timestamp := time.Unix(0, int64(float64(time.Second)**e.Time))
	return event.NewWithProperties(eventType, event.COLLECTD, dimensions, properties, timestamp)
}
