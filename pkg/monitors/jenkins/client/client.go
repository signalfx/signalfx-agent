package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/bndr/gojenkins"
)

type JenkinsClient struct {
	Host       string
	MetricsKey string
	Port       string
	Scheme     string
	HTTPClient *http.Client
	GoJenkins  *gojenkins.Jenkins
}

func NewJenkinsClient(host string, metricKey string, port string, scheme string, client *http.Client) (JenkinsClient, error) {
	jenkins := gojenkins.CreateJenkins(client, fmt.Sprintf("%s://%s:%s/", scheme, host, port))
	_, err := jenkins.Init()
	if err != nil {
		return JenkinsClient{}, err
	}

	return JenkinsClient{
		Host:       host,
		MetricsKey: metricKey,
		Port:       port,
		Scheme:     scheme,
		HTTPClient: client,
		GoJenkins:  jenkins,
	}, nil
}

// FetchJSON builds the URL of the endpoint and fetches the json envelope
// and deserialize it into the obj
func (c *JenkinsClient) FetchJSON(endpoint string, obj interface{}) error {
	url := fmt.Sprintf("%s://%s:%s/%s", c.Scheme, c.Host, c.Port, endpoint)
	res, err := fetchResponse(url, c.HTTPClient)
	if err != nil {
		return err
	}
	defer res.Close()

	err = json.NewDecoder(res).Decode(obj)
	if err != nil {
		return fmt.Errorf("could not get url %s: %v", url, err)
	}
	return nil
}

// FetchText builds the URL of the endpoint and fetches the text into result
func (c *JenkinsClient) FetchText(endpoint string) (string, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.Scheme, c.Host, c.Port, endpoint)
	res, err := fetchResponse(url, c.HTTPClient)
	if err != nil {
		return "", err
	}
	defer res.Close()

	responseData, err := ioutil.ReadAll(res)
	if err != nil {
		return "", err
	}
	return string(responseData), nil
}

// fetchResponse takes a URL and HTTP client and returns a reader
// caller should always close the reader
func fetchResponse(url string, c *http.Client) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build request for url %s: %v", url, err)
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not get url %s: %v", url, err)
	}
	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		res.Body.Close()
		return nil, fmt.Errorf("received status code that's not 200: %s , url: %s , body: %s", res.Status, url, string(body))
	}
	return res.Body, nil
}
