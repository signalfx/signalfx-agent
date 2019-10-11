package hostid

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// AWSUniqueID constructs the unique EC2 instance of the underlying host.  If
// not running on EC2, returns the empty string.
func AWSUniqueID() string {
	c := http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := c.Get("http://169.254.169.254/2014-11-05/dynamic/instance-identity/document")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var doc struct {
		AccountID  string `json:"accountId"`
		InstanceID string `json:"instanceId"`
		Region     string `json:"region"`
	}

	err = json.Unmarshal(body, &doc)
	if err != nil {
		return ""
	}

	if doc.AccountID == "" || doc.InstanceID == "" || doc.Region == "" {
		return ""
	}

	return fmt.Sprintf("%s_%s_%s", doc.InstanceID, doc.Region, doc.AccountID)
}
