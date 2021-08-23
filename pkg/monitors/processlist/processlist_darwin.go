//go:build darwin
// +build darwin

package processlist

type osCache struct {
}

func initOSCache() *osCache {
	return &osCache{}
}

func ProcessList(conf *Config, cache *osCache) ([]*TopProcess, error) {
	return nil, nil
}
