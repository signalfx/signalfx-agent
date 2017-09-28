package kubelet

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func augmentCertPoolFromCAFile(basePool *x509.CertPool, caCertPath string) bool {
	bytes, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"caCertPath": caCertPath,
		}).Error("CA cert path could not be read")
		return false
	}

	if !basePool.AppendCertsFromPEM(bytes) {
		log.WithFields(log.Fields{
			"caCertPath": caCertPath,
		}).Error("CA cert file is not the right format")
		return false
	}

	return true
}

// An http transport that injects an OAuth bearer token onto each request
type transportWithToken struct {
	*http.Transport
	token string
}

// Override the only method that the client actually calls on the transport to
// do the request.
func (t *transportWithToken) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("bearer %s", t.token))
	return t.Transport.RoundTrip(req)
}
