package config

import (
	"time"

	"github.com/signalfx/signalfx-agent/lib/correlations"
)

func ClientConfigFromWriterConfig(conf *WriterConfig) correlations.ClientConfig {
	return correlations.ClientConfig{
		Config: correlations.Config{
			MaxRequests:         conf.PropertiesMaxRequests,
			MaxBuffered:         conf.PropertiesMaxBuffered,
			MaxRetries:          conf.TraceHostCorrelationMaxRequestRetries,
			LogDimensionUpdates: conf.LogDimensionUpdates,
			SendDelay:           time.Duration(conf.PropertiesSendDelaySeconds) * time.Second,
			PurgeInterval:       conf.TraceHostCorrelationPurgeInterval.AsDuration(),
		},
		AccessToken: conf.SignalFxAccessToken,
		URL:         conf.ParsedAPIURL(),
	}
}
