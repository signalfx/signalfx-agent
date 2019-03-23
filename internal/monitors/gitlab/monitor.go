package gitlab

import (
	"github.com/signalfx/signalfx-agent/internal/monitors"
	pe "github.com/signalfx/signalfx-agent/internal/monitors/prometheusexporter"
)

func init() {
	monitors.Register("gitlab", func() interface{} {
		return &pe.Monitor{IncludedMetrics: gitlabIncludedMetrics}
	}, &pe.Config{})

	monitors.Register("gitlab-runner", func() interface{} {
		return &pe.Monitor{IncludedMetrics: gitlabIncludedMetrics}
	}, &pe.Config{})

	monitors.Register("gitlab-gitaly", func() interface{} {
		return &pe.Monitor{IncludedMetrics: gitlabIncludedMetrics, ExtraDimensions: map[string]string{
			"metric_source": "gitlab-gitaly"}}
	}, &pe.Config{})

	monitors.Register("gitlab-sidekiq", func() interface{} {
		return &pe.Monitor{IncludedMetrics: gitlabIncludedMetrics}
	}, &pe.Config{})

	monitors.Register("gitlab-workhorse", func() interface{} {
		return &pe.Monitor{IncludedMetrics: gitlabIncludedMetrics}
	}, &pe.Config{})

	// Send all unicorn metrics
	monitors.Register("gitlab-unicorn", func() interface{} { return &pe.Monitor{} }, &pe.Config{
		MetricPath: "/-/metrics",
	})
}
