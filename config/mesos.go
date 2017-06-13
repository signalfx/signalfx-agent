package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

// MesosID - struct for storing mesos id
type MesosID struct {
	ID string
}

// MesosphereClient - client for interacting with mesos api
type MesosClient struct {
	Config   *viper.Viper
	hostURL  string
	hostPort int
	client   http.Client
}

// NewMesosClient client
func NewMesosClient(config *viper.Viper) (*MesosClient, error) {
	return &MesosClient{config, "", 5051, http.Client{}}, nil
}

// GetID - retrieves the mesos id for the node
func (mesos *MesosClient) GetID() (*MesosID, error) {
	resp, err := mesos.client.Get(fmt.Sprintf("%s/state", mesos.hostURL))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get task states: (code=%d, body=%s)",
			resp.StatusCode, body[:512])
	}

	id := &MesosID{}
	if err := json.Unmarshal(body, id); err != nil {
		return nil, err
	}

	return id, nil
}

// Configure the mesosphere observer/client
func (mesos *MesosClient) Configure(config *viper.Viper, port int) error {
	mesos.Config = config
	mesos.hostPort = port
	return mesos.load()
}

func (mesos *MesosClient) load() error {
	if hostname, err := os.Hostname(); err == nil {
		mesos.Config.SetDefault("hosturl", fmt.Sprintf("http://%s:%d", hostname, mesos.hostPort))
	}

	hostURL := mesos.Config.GetString("hosturl")

	if len(hostURL) == 0 {
		return errors.New("hostURL config value missing")
	}
	mesos.hostURL = hostURL

	mesos.client = http.Client{
		Timeout: 10 * time.Second,
	}
	return nil
}
