package hostid

import (
        "encoding/json"
        "fmt"
        "io/ioutil"
        "net/http"
        "time"
)

// AzureUniqueID constructs the unique ID of the underlying Azure VM.  If
// not running on Azure VM, returns the empty string.
func AzureUniqueID() string {
  c := http.Client{
    Timeout: 1 * time.Second,
  }
  req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance?api-version=2017-08-01", nil)
  if err!= nil {
    return ""
  }

  req.Header.Set("Metadata", "true")
  resp, err := c.Do(req)
  if err!= nil {
    return ""
  }

  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return ""
  }

  type Info struct {
    SubscriptionID string `json:"subscriptionId"`
    ResourceGroupName string `json:"resourceGroupName"`
    Name string `json:"name"`
  }

  var compute struct {
    Doc Info `json:"compute"`
  }

  err = json.Unmarshal(body, &compute)
  if err != nil {
    return ""
  }

  if compute.Doc.SubscriptionID == "" || compute.Doc.ResourceGroupName == "" || compute.Doc.Name == "" {
    return ""
  }

  return fmt.Sprintf("%s/%s/microsoft.compute/virtualmachines/%s", compute.Doc.SubscriptionID, compute.Doc.ResourceGroupName, compute.Doc.Name)
}
