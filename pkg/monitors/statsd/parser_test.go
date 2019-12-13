package statsd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFields(t *testing.T) {
	cases := []struct {
		pattern        string
		substrs        []string
		startWithField bool
		expectNil      bool
	}{
		{
			"metric.count",
			[]string{"metric.count"},
			false,
			false,
		},
		{
			"metric.count{",
			nil,
			false,
			true,
		},
		{
			"{metric.count",
			nil,
			false,
			true,
		},
		{
			"metric.count}",
			nil,
			false,
			true,
		},
		{
			"{{metric.count}}",
			nil,
			false,
			true,
		},
		{
			"{metric.count}",
			[]string{"metric.count"},
			true,
			false,
		},
		{
			"cluster.cds_{traffic}_{mesh}_{service}-vn_{}.{action}",
			[]string{"cluster.cds_", "traffic", "_", "mesh", "_", "service", "-vn_", "", ".", "action"},
			false,
			false,
		},
		{
			"cluster.cds_{traffic}_{mesh}_{service}-vn_{}.{action}-prod",
			[]string{"cluster.cds_", "traffic", "_", "mesh", "_", "service", "-vn_", "", ".", "action", "-prod"},
			false,
			false,
		},
		{
			"{cluster}.cds_{traffic}_{mesh}_{service}-vn_{}.{action}",
			[]string{"cluster", ".cds_", "traffic", "_", "mesh", "_", "service", "-vn_", "", ".", "action"},
			true,
			false,
		},
		{
			"{cluster}.cds_{traffic}_{mesh}_{service}-vn_{}.{action}-dev",
			[]string{"cluster", ".cds_", "traffic", "_", "mesh", "_", "service", "-vn_", "", ".", "action", "-dev"},
			true,
			false,
		},
		{
			// Cannot have back-to-back patterns
			"{cluster}.cds_{traffic}{mesh}_{service}-vn_{}.{action}",
			nil,
			false,
			true,
		},
	}
	for i := range cases {
		tt := cases[i]
		t.Run(tt.pattern, func(t *testing.T) {
			fp := parseFields(tt.pattern)
			if tt.expectNil {
				require.Nil(t, fp)
				return
			}

			require.Equal(t, fieldPattern{substrs: tt.substrs, startWithField: tt.startWithField}, *fp)
		})
	}
}
