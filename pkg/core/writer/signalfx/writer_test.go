package signalfx

import (
	"testing"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/stretchr/testify/require"
)

var essentialWriterConfig = config.WriterConfig{
	SignalFxAccessToken:                 "11111",
	PropertiesHistorySize:               100,
	PropertiesSendDelaySeconds:          1,
	TraceExportFormat:                   "zipkin",
	TraceHostCorrelationMetricsInterval: timeutil.Duration(1 * time.Second),
	TraceHostCorrelationPurgeInterval:   timeutil.Duration(1 * time.Second),
	StaleServiceTimeout:                 timeutil.Duration(1 * time.Second),
	EventSendIntervalSeconds:            1,
}

func TestWriterSetup(t *testing.T) {
	t.Run("Overrides event URL", func(t *testing.T) {
		t.Parallel()
		conf := essentialWriterConfig
		conf.EventEndpointURL = "http://example.com/v2/event"
		writer, err := New(&conf, nil, nil, nil, nil, nil)

		require.Nil(t, err)
		require.Equal(t, "http://example.com/v2/event", writer.client.EventEndpoint)
	})

	t.Run("Sets default event URL", func(t *testing.T) {
		t.Parallel()
		conf := essentialWriterConfig
		conf.IngestURL = "http://example.com"
		writer, err := New(&conf, nil, nil, nil, nil, nil)
		require.Nil(t, err)
		require.Equal(t, "http://example.com/v2/event", writer.client.EventEndpoint)
	})
}
