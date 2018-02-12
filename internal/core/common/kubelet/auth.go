package kubelet

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

func augmentCertPoolFromCAFile(basePool *x509.CertPool, caCertPath string) error {
	bytes, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return errors.Wrapf(err, "CA cert path %s could not be read", caCertPath)
	}

	if !basePool.AppendCertsFromPEM(bytes) {
		return errors.Errorf("CA cert file %s is not the right format", caCertPath)
	}

	return nil
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
