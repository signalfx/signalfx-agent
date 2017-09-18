package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
)

// DimProperties represents a set of properties associated with a given
// dimension value
type DimProperties struct {
	Dimension
	// Properties to be set on the dimension
	Properties map[string]string
}

// Dimension represents a specific dimension value
type Dimension struct {
	// Name of the dimension
	Name string
	// Value of the dimension
	Value string
}

type dimensionPropertyClient struct {
	client *http.Client
	Token  string
	// Keeps track of what has been synced so we don't do unnecessary syncs
	history map[Dimension]map[string]string
}

func newDimensionPropertyClient() *dimensionPropertyClient {
	return &dimensionPropertyClient{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		history: make(map[Dimension]map[string]string),
	}
}

// SetPropertiesOnDimension will set custom properties on a specific dimension
// value.  It will wipe out any description or tags on the dimension.
func (dpc *dimensionPropertyClient) SetPropertiesOnDimension(dimProps *DimProperties) error {
	prev := dpc.history[dimProps.Dimension]
	if !reflect.DeepEqual(prev, dimProps.Properties) {
		log.WithFields(log.Fields{
			"name":  dimProps.Name,
			"value": dimProps.Value,
			"props": dimProps.Properties,
		}).Debug("Syncing properties to dimension")

		err := dpc.doReq(dimProps.Name, dimProps.Value, dimProps.Properties)
		if err != nil {
			return err
		}
		dpc.history[dimProps.Dimension] = dimProps.Properties
	}
	return nil
}

func (dpc *dimensionPropertyClient) doReq(key, value string, props map[string]string) error {
	json, err := json.Marshal(map[string]interface{}{
		"key":              key,
		"value":            value,
		"customProperties": props,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("https://api.signalfx.com/v2/dimension/%s/%s", key, value),
		bytes.NewReader(json))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SF-TOKEN", dpc.Token)

	resp, err := dpc.client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code %d on response %s", resp.StatusCode, string(body))
	}

	return err
}
