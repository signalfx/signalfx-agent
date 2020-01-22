package conviva

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// responseError for Conviva error response
type responseError struct {
	Message string
	Code    int64
	Request string
	Reason  string
}

// httpClient interface to provide for Conviva API specific implementation
type httpClient interface {
	get(ctx context.Context, v interface{}, url string) (int, error)
}

type convivaHTTPClient struct {
	client *http.Client
}

// newConvivaClient factory function for creating HTTPClientt
func newConvivaClient(client *http.Client) httpClient {
	return &convivaHTTPClient{
		client: client,
	}
}

// Get method for Conviva API specific gets
func (c *convivaHTTPClient) get(ctx context.Context, v interface{}, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != 200 {
		responseError := responseError{}
		if err := json.Unmarshal(body, &responseError); err == nil {
			return res.StatusCode, fmt.Errorf("%+v", responseError)
		}
		return res.StatusCode, fmt.Errorf("%+v", res)
	}
	err = json.Unmarshal(body, v)
	return res.StatusCode, err
}
