package redis

import (
	"errors"
	"strconv"
	"strings"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
)

func parseInfoString(infoStr string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(infoStr, "\r\n") {
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			logger.Warnf("Non-blank/comment info line is not in form <key>:<value>", line)
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}

func metricsFromData(infoMap map[string]string, extraMetrics map[string]bool) ([]*datapoint.Datapoint, error) {
	var out []*datapoint.Datapoint
	for k, v := range infoMap {
		if !nonCustomMetrics[k] && !extraMetrics[k] {
			continue
		}

		if strings.HasPrefix(k, "db") {
			dps, err := makeKeyspaceMetrics(k, v)
			if err != nil {
				logger.WithError(err).Warnf("Could not construct keyspace metrics from %s:%s", k, v)
				continue
			}
			out = append(out, dps...)
		} else {
			dp, err := makeRegularMetric(k, v)
			if err != nil {
				logger.WithError(err).Warnf("Could not construct metric from %s:%s", k, v)
				continue
			}
			out = append(out, dp)
		}
	}
	return out, nil
}

func makeMetadataDatapoint(infoMap map[string]string) (*datapoint.Datapoint, error) {
	ver := infoMap["redis_version"]
	if ver == "" {
		return nil, errors.New("No Redis version available")
	}

	return sfxclient.Gauge("redis.metadata", map[string]string{"redis_version": ver}, 1), nil
}

func makeRegularMetric(k, v string) (*datapoint.Datapoint, error) {
	metricName := "redis." + k

	var dp *datapoint.Datapoint
	if strings.Contains(v, ".") {
		asFloat, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		if cumulativeMetrics[k] {
			dp = sfxclient.CumulativeF(metricName, nil, asFloat)
		} else {
			dp = sfxclient.GaugeF(metricName, nil, asFloat)
		}
	} else {
		asInt, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		if cumulativeMetrics[k] {
			dp = sfxclient.Cumulative(metricName, nil, asInt)
		} else {
			dp = sfxclient.Gauge(metricName, nil, asInt)
		}
	}
	return dp, nil
}

func makeKeyspaceMetrics(k, v string) ([]*datapoint.Datapoint, error) {
	metrics := strings.Split(v, ",")
	dbNum := strings.TrimPrefix(k, "db")

	out := make([]*datapoint.Datapoint, 0, len(metrics))
	for _, m := range metrics {
		parts := strings.Split(m, "=")
		if len(parts) != 2 {
			logger.Warnf("Keyspace info %s has invalid metric part %s", v, m)
			continue
		}
		asInt, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			logger.Warnf("Keyspace info %s for db %s has non-integer value: %s", parts[0], dbNum, parts[1])
			continue
		}
		out = append(out, sfxclient.Gauge("redis.db."+parts[0], map[string]string{"db": dbNum}, asInt))
	}
	return out, nil
}
