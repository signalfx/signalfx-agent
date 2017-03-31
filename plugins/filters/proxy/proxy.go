package proxy

import (
	"log"
	"os"
	"strings"

	"github.com/signalfx/neo-agent/plugins"
	"github.com/signalfx/neo-agent/services"
	"github.com/spf13/viper"
)

const (
	pluginType = "filters/proxy"
	httpProxy = "http_proxy"
	httpsProxy = "http_proxy"
	noProxy = "no_proxy"
)

// Proxy manages the proxy environment variables
type Proxy struct {
	plugins.Plugin
	http string
	https string
	skip string
}

func init() {
	plugins.Register(pluginType, NewProxy)
}

// NewProxy creates a new instance
func NewProxy(name string, config *viper.Viper) (plugins.IPlugin, error) {
	plugin, err := plugins.NewPlugin(name, pluginType, config)
	if err != nil {
		return nil, err
	}
	return &Proxy{plugin, config.GetString("http"), config.GetString("https"), config.GetString("skip")}, nil
}

func setEnvVar(key string, value string, logMessage bool) {
	if logMessage {
		log.Printf("setting %s to %s", key, value)
	}
	os.Setenv(strings.ToLower(key), value)
	os.Setenv(strings.ToUpper(key), value)
}

// Reload the config and environment variables
func (proxy *Proxy) Reload(config *viper.Viper) error {
	log.Println("reloading proxy filter")

	proxy.Config = config
	proxy.http = config.GetString("http")
	proxy.https = config.GetString("https")
	proxy.skip = config.GetString("skip")

	if len(proxy.http) > 0 {
		setEnvVar(httpProxy, proxy.http, true)
	}

	if len(proxy.https) > 0 {
		setEnvVar(httpsProxy, proxy.https, true)
	}

	if len(proxy.skip) > 0 {
		setEnvVar(noProxy, proxy.skip, true)
	}

	return nil
}

// Start sets the proxy related environment variables
func (proxy *Proxy) Start() (err error) {
	log.Println("starting proxy filter")

	if len(proxy.http) > 0 {
		setEnvVar(httpProxy, proxy.http, true)
	}

	if len(proxy.https) > 0 {
		setEnvVar(httpsProxy, proxy.https, true)
	}

	if len(proxy.skip) > 0 {
		setEnvVar(noProxy, proxy.skip, true)
	}

	return nil
}

// Stop resets no_proxy environment variable
func (proxy *Proxy) Stop() {
	log.Println("stopping proxy filter")
	if len(proxy.skip) > 0 {
		setEnvVar(noProxy, proxy.skip, true)
	}
}

// Map sets the service IPs in the no_proxy environment variable
func (proxy *Proxy) Map(instances services.Instances) (services.Instances, error) {
	if (len(proxy.http) > 0) || (len(proxy.https) > 0) {
		noProxyList := []string{}
		for _, skipItem := range strings.Split(proxy.skip, ",") {
			if len(skipItem) > 0 {
				noProxyList = append(noProxyList, skipItem)
			}
		}
		for _, instance := range instances {
			if len(instance.Port.IP) > 0 {
				noProxyList = append(noProxyList, instance.Port.IP)
			}
		}
		setEnvVar(noProxy, strings.Join(noProxyList, ","), false)
	}
	return instances, nil
}
