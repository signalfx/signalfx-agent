package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	nodeStatsEndpoint          = "_nodes/_local/stats/transport,http,process,jvm,indices,thread_pool"
	clusterHealthStatsEndpoint = "_cluster/health"
	nodeInfoEndpoint           = "_nodes/_local"
	masterNodeEndpoint         = "_cluster/state/master_node"
	allIndexStatsEndpoint      = "_all/_stats"
)

type esClient struct {
	host       string
	port       string
	scheme     string
	username   string
	password   string
	httpClient *http.Client
}

// ESHttpClient holds methods hitting various ES stats endpoints
type ESHttpClient interface {
	GetNodeAndThreadPoolStats() (*NodeStatsOutput, error)
	GetClusterStats() (*ClusterStatsOutput, error)
	GetNodeInfo() (*NodeInfoOutput, error)
	GetMasterNodeInfo() (*MasterInfoOutput, error)
	GetIndexStats() (*IndexStatsOutput, error)
}

// NewESClient creates a new esClient
func NewESClient(host string, port string, useHTTPS bool, skipVerify bool, username string, password string) ESHttpClient {
	scheme := "http"
	httpClient := &http.Client{}

	if useHTTPS {
		scheme = "https"

		if skipVerify {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			httpClient.Transport = transport
		}
	}

	return &esClient{
		host:       host,
		port:       port,
		scheme:     scheme,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}
}

// Method to collect index stats
func (c *esClient) GetIndexStats() (*IndexStatsOutput, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.scheme, c.host, c.port, allIndexStatsEndpoint)

	var indexStatsOutput IndexStatsOutput

	err := c.fetchJSON(url, &indexStatsOutput)

	if err != nil {
		return nil, err
	}

	return &indexStatsOutput, nil
}

// Method to identify the master node
func (c *esClient) GetMasterNodeInfo() (*MasterInfoOutput, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.scheme, c.host, c.port, masterNodeEndpoint)

	var masterInfoOutput MasterInfoOutput

	err := c.fetchJSON(url, &masterInfoOutput)

	if err != nil {
		return nil, err
	}

	return &masterInfoOutput, nil
}

// Method to fetch node info
func (c *esClient) GetNodeInfo() (*NodeInfoOutput, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.scheme, c.host, c.port, nodeInfoEndpoint)

	var nodeInfoOutput NodeInfoOutput

	err := c.fetchJSON(url, &nodeInfoOutput)

	if err != nil {
		return nil, err
	}

	return &nodeInfoOutput, nil
}

// Method to fetch cluster stats
func (c *esClient) GetClusterStats() (*ClusterStatsOutput, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.scheme, c.host, c.port, clusterHealthStatsEndpoint)

	var clusterStatsOutput ClusterStatsOutput

	err := c.fetchJSON(url, &clusterStatsOutput)

	if err != nil {
		return nil, err
	}

	return &clusterStatsOutput, nil
}

// Method to fetch node stats
func (c *esClient) GetNodeAndThreadPoolStats() (*NodeStatsOutput, error) {
	url := fmt.Sprintf("%s://%s:%s/%s", c.scheme, c.host, c.port, nodeStatsEndpoint)

	var nodeStatsOutput NodeStatsOutput

	err := c.fetchJSON(url, &nodeStatsOutput)

	if err != nil {
		return nil, err
	}

	return &nodeStatsOutput, nil
}

// Fetches a JSON response and puts it into an object
func (c *esClient) fetchJSON(url string, obj interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("could not get url %s: %v", url, err)
	}

	req.SetBasicAuth(c.username, c.password)
	res, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("could not get url %s: %v", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("received status code that's not 200: %s, url: %s", res.Status, url)
	}

	err = json.NewDecoder(res.Body).Decode(obj)

	if err != nil {
		return fmt.Errorf("could not get url %s: %v", url, err)
	}

	return nil
}
