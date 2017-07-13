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
	httpProxy  = "http_proxy"
	httpsProxy = "https_proxy"
	noProxy    = "no_proxy"
)

// Proxy manages the proxy environment variables
type Proxy struct {
	http  string
	https string
	skip  string
}

func init() {
	plugins.Register(pluginType, func() interface{} { return &Proxy{} })
}

func setEnvVar(key string, value string, logMessage bool) {
	if logMessage {
		log.Printf("setting %s to %s", key, value)
	}
	os.Setenv(strings.ToLower(key), value)
	os.Setenv(strings.ToUpper(key), value)
}

// Configure the proxy environment variables
func (proxy *Proxy) Configure(config *viper.Viper) error {
	log.Println("reloading proxy filter")

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

// Shutdown resets the no_proxy environment variable
func (proxy *Proxy) Shutdown() {
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
