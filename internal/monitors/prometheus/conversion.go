package prometheus

import (
	"fmt"

	dto "github.com/prometheus/client_model/go"
	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/golib/sfxclient"
	"github.com/signalfx/signalfx-agent/internal/utils"
)

type extractor func(m *dto.Metric) float64
type dpFactory func(string, map[string]string, float64) *datapoint.Datapoint

func gaugeExtractor(m *dto.Metric) float64 {
	return m.GetGauge().GetValue()
}

func untypedExtractor(m *dto.Metric) float64 {
	return m.GetUntyped().GetValue()
}

func counterExtractor(m *dto.Metric) float64 {
	return m.GetCounter().GetValue()
}

func convertMetricFamily(mf *dto.MetricFamily) []*datapoint.Datapoint {
	if mf.Type == nil || mf.Name == nil {
		return nil
	}
	switch *mf.Type {
	case dto.MetricType_GAUGE:
		return makeSimpleDatapoints(*mf.Name, mf.Metric, sfxclient.GaugeF, gaugeExtractor)
	case dto.MetricType_COUNTER:
		return makeSimpleDatapoints(*mf.Name, mf.Metric, sfxclient.CumulativeF, counterExtractor)
	case dto.MetricType_UNTYPED:
		return makeSimpleDatapoints(*mf.Name, mf.Metric, sfxclient.GaugeF, untypedExtractor)
	case dto.MetricType_SUMMARY:
		return makeSummaryDatapoints(*mf.Name, mf.Metric)
	// TODO: figure out how to best convert histograms, in particular the
	// upper bound value
	case dto.MetricType_HISTOGRAM:
		return nil
	default:
		return nil
	}
}

func makeSimpleDatapoints(name string, ms []*dto.Metric, dpf dpFactory, e extractor) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, m := range ms {
		dps = append(dps, dpf(name, labelsToDims(m.Label), e(m)))
	}
	return dps
}

func makeSummaryDatapoints(name string, ms []*dto.Metric) []*datapoint.Datapoint {
	var dps []*datapoint.Datapoint
	for _, m := range ms {
		dims := labelsToDims(m.Label)
		s := m.GetSummary()
		if s == nil {
			continue
		}

		if s.SampleCount != nil {
			dps = append(dps, sfxclient.Gauge(name, dims, int64(s.GetSampleCount())))
		}

		if s.SampleSum != nil {
			dps = append(dps, sfxclient.GaugeF(name, dims, s.GetSampleSum()))
		}

		qs := s.GetQuantile()
		for i := range qs {
			quantileDims := utils.MergeStringMaps(dims, map[string]string{
				"quantile": fmt.Sprintf("%f", qs[i].GetQuantile()),
			})
			dps = append(dps, sfxclient.GaugeF(name, quantileDims, qs[i].GetValue()))
		}
	}
	return dps
}

func labelsToDims(labels []*dto.LabelPair) map[string]string {
	dims := map[string]string{}
	for i := range labels {
		dims[labels[i].GetName()] = labels[i].GetValue()
	}
	return dims
}
