package hostid

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/signalfx/signalfx-agent/pkg/utils/timeutil"
	log "github.com/sirupsen/logrus"
)

// AWSUniqueID constructs the unique EC2 instance of the underlying host.  If
// not running on EC2, returns the empty string.
func AWSUniqueID(cloudMetadataTimeout timeutil.Duration) string {
	c := http.Client{
		Timeout: cloudMetadataTimeout.AsDuration(),
	}

	resp, err := c.Get("http://169.254.169.254/2014-11-05/dynamic/instance-identity/document")
	if err != nil {
		log.WithFields(log.Fields{
			"detail": err,
		}).Info("No AWS metadata server detected, assuming not on EC2")
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		log.Debug("HTTP status 404, assuming not on EC2")
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Info("Failed to read AWS instance-identity response")
		return ""
	}

	var doc struct {
		AccountID  string `json:"accountId"`
		InstanceID string `json:"instanceId"`
		Region     string `json:"region"`
	}

	err = json.Unmarshal(body, &doc)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  string(body),
		}).Info("Failed to unmarshal AWS instance-identity response")
		return ""
	}

	if doc.AccountID == "" || doc.InstanceID == "" || doc.Region == "" {
		log.Errorf("One (or more) required field is empty. AccountID: %s ; InstanceID: %s ; Region: %s", doc.AccountID, doc.InstanceID, doc.Region)
		return ""
	}

	return fmt.Sprintf("%s_%s_%s", doc.InstanceID, doc.Region, doc.AccountID)
}
