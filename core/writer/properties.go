package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/signalfx/neo-agent/monitors/types"
	log "github.com/sirupsen/logrus"
)

type dimensionPropertyClient struct {
	client *http.Client
	Token  string
	APIURL *url.URL
	// Keeps track of what has been synced so we don't do unnecessary syncs
	history map[types.Dimension]map[string]string
}

func newDimensionPropertyClient() *dimensionPropertyClient {
	return &dimensionPropertyClient{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		history: make(map[types.Dimension]map[string]string),
	}
}

// SetPropertiesOnDimension will set custom properties on a specific dimension
// value.  It will wipe out any description or tags on the dimension.
func (dpc *dimensionPropertyClient) SetPropertiesOnDimension(dimProps *types.DimProperties) error {
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

	url, err := dpc.APIURL.Parse(fmt.Sprintf("/v2/dimension/%s/%s", key, value))
	if err != nil {
		return errors.Wrapf(err, "Could not construct dimension property PUT URL with %s / %s", key, value)
	}

	req, err := http.NewRequest(
		"PUT",
		url.String(),
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
