package conviva

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ErrorResponse for Conviva error response
type ErrorResponse struct {
	Message string
	Code    float64
	Request string
	Reason  string
}

// HTTPClient interface to provide for Conviva API specific implementation
type HTTPClient interface {
	Get(ctx context.Context, v interface{}, url string) error
}

type convivaHTTPClient struct {
	client  *http.Client
	username string
	password string
}

// NewConvivaClient factory function for creating HTTPClientt
func NewConvivaClient(client *http.Client, username string, password string) HTTPClient {
	return &convivaHTTPClient{
			client:   client,
			username: username,
			password: password,
	}
}

// Get method for Conviva API specific gets
func (c *convivaHTTPClient) Get(ctx context.Context, v interface{}, url string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.SetBasicAuth(c.username, c.password)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200  {
		errorResponse := ErrorResponse{}
		if err := json.Unmarshal(body, &errorResponse); err == nil {
			return fmt.Errorf("%+v", errorResponse)
		}
		return fmt.Errorf("%+v", res)
	}
	err = json.Unmarshal(body, v)
	return err
}