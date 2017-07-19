package monitors

import (
	"os"
	"strings"

	"github.com/signalfx/neo-agent/observers"
)

const (
	noProxy = "no_proxy"
)

func setNoProxyEnvvar(value string) {
	os.Setenv("no_proxy", value)
	os.Setenv("NO_PROXY", value)
}

func isProxying() bool {
	return os.Getenv("http_proxy") != "" || os.Getenv("https_proxy") != "" ||
		os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != ""
}

func getNoProxyEnvvar() string {
	noProxy := os.Getenv("NO_PROXY")
	if noProxy == "" {
		noProxy = os.Getenv("no_proxy")
	}
	return noProxy
}

// DisableServices sets the service IPs in the no_proxy environment variable
func EnsureProxyingDisabledForService(service *observers.ServiceInstance) {
	if isProxying() && len(service.Port.IP) > 0 {
		serviceIP := service.Port.IP
		noProxy := getNoProxyEnvvar()

		for _, existingIP := range strings.Split(noProxy, ",") {
			if existingIP == serviceIP {
				return
			}
		}

		setNoProxyEnvvar(noProxy + "," + serviceIP)
	}
}
