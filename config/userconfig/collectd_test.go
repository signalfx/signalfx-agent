package userconfig

import "testing"

func TestCollectd_Parse(t *testing.T) {
	var one = int(1)
	var two = int(2)
	var three = int(3)
	var four = int(4)
	var five = int(5)
	var f = false

	var loadedConfig = Collectd{
		Interval:             &one,
		Timeout:              &two,
		ReadThreads:          &three,
		WriteQueueLimitHigh:  &four,
		WriteQueueLimitLow:   &five,
		CollectInternalStats: &f,
	}
	var parsedConfig = map[string]interface{}{
		"interval":             1,
		"timeout":              2,
		"readThreads":          3,
		"writeQueueLimitHigh":  4,
		"writeQueueLimitLow":   5,
		"collectInternalStats": false,
	}

	type args struct {
		file string
	}
	type want struct {
		collectd Collectd
		parsed   map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{"parsed collectd config matched", args{"testdata/collectd/matched.yaml"}, want{loadedConfig, parsedConfig}, false},
		{"parsed collectd config mismatched", args{"testdata/collectd/mismatched.yaml"}, want{loadedConfig, parsedConfig}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c = &Collectd{}

			if err := c.LoadYAML(tt.args.file); err != nil {
				t.Errorf("%v", err)
			}

			// Parse the configuration
			var parsedConfigurations = map[string]interface{}{}
			if err := c.Parse(parsedConfigurations); err != nil {
				t.Errorf("Collectd.Parse() error = %v", err)
			}

			var mismatched = false
			var key string
			for k, v := range tt.want.parsed {
				if parsedConfigurations[k] != v {
					mismatched = true
					key = k
				}
			}

			if mismatched != tt.wantErr {
				t.Errorf("Collectd.Parse() Configuration %s: parsed value %v does not match expected value %v", key, parsedConfigurations[key], tt.want.parsed[key])
			}
		})
	}
}
