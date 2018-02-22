package cadvisor

// Parts of this module are copied from the heapster project, specifically the
// file https://github.com/kubernetes/heapster/blob/master/metrics/sources/kubelet/kubelet_client.go
// We can't just import the heapster project because it depends on the main K8s
// codebase which breaks a lot of stuff if we try and import it transitively
// alongside the k8s client-go library.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	info "github.com/google/cadvisor/info/v1"
	"github.com/signalfx/signalfx-agent/internal/core/common/kubelet"
	"github.com/signalfx/signalfx-agent/internal/core/config"
	"github.com/signalfx/signalfx-agent/internal/monitors"
	"github.com/signalfx/signalfx-agent/internal/monitors/types"
	log "github.com/sirupsen/logrus"
)

const (
	kubeletStatsType = "kubelet-stats"
)

// MONITOR(kubelet-stats): This monitor pulls cadvisor metrics through a
// Kubernetes kubelet instance via the /stats/container endpoint.

func init() {
	monitors.Register(kubeletStatsType, func() interface{} { return &KubeletStatsMonitor{} }, &KubeletStatsConfig{})
}

// KubeletStatsConfig respresents config for the Kubelet stats monitor
type KubeletStatsConfig struct {
	config.MonitorConfig
	Config
	KubeletAPI   kubelet.APIConfig `yaml:"kubeletAPI" default:"{}"`
	LogResponses bool              `yaml:"logResponses" default:"false"`
}

// KubeletStatsMonitor will pull container metrics from the /stats/ endpoint of
// the Kubelet API.  This is the same thing that other K8s metric solutions
// like Heapster use and should eventually completely replace out use of the
// cAdvisor endpoint that some K8s deployments expose.  Right now, this assumes
// a certain format of the stats that come off of the endpoints.  TODO: Figure
// out if this is versioned and how to access versioned endpoints.
type KubeletStatsMonitor struct {
	Monitor
	Output types.Output
}

// Configure the Kubelet Stats monitor
func (ks *KubeletStatsMonitor) Configure(conf *KubeletStatsConfig) error {
	kubeletAPI := conf.KubeletAPI
	if kubeletAPI.URL == "" {
		kubeletAPI.URL = fmt.Sprintf("https://%s:10250", ks.AgentMeta.Hostname)
	}
	client, err := kubelet.NewClient(&kubeletAPI)
	if err != nil {
		return err
	}

	return ks.Monitor.Configure(&conf.Config, &conf.MonitorConfig, ks.Output.SendDatapoint,
		newKubeletInfoProvider(client, conf.LogResponses))
}

type statsRequest struct {
	// The name of the container for which to request stats.
	// Default: /
	ContainerName string `json:"containerName,omitempty"`

	// Max number of stats to return.
	// If start and end time are specified this limit is ignored.
	// Default: 60
	NumStats int `json:"num_stats,omitempty"`

	// Start time for which to query information.
	// If omitted, the beginning of time is assumed.
	Start time.Time `json:"start,omitempty"`

	// End time for which to query information.
	// If omitted, current time is assumed.
	End time.Time `json:"end,omitempty"`

	// Whether to also include information from subcontainers.
	// Default: false.
	Subcontainers bool `json:"subcontainers,omitempty"`
}

type kubeletInfoProvider struct {
	client       *kubelet.Client
	lastUpdate   time.Time
	logResponses bool
}

func (kip *kubeletInfoProvider) SubcontainersInfo(containerName string) ([]info.ContainerInfo, error) {
	curTime := time.Now()
	containers, err := kip.getAllContainers(kip.lastUpdate, curTime)
	if err != nil {
		return nil, err
	}

	if len(containers) > 0 {
		kip.lastUpdate = curTime
	}

	return filterPodContainers(containers), nil
}

func filterPodContainers(containers []info.ContainerInfo) []info.ContainerInfo {
	out := make([]info.ContainerInfo, 0)
	for _, c := range containers {
		// Only get containers that are in pods
		if c.Spec.Labels != nil || len(c.Spec.Labels["io.kubernetes.pod.uid"]) > 0 {
			out = append(out, c)
		}
	}
	return out
}

func (kip *kubeletInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	req, err := kip.client.NewRequest("GET", "/spec/", nil)
	if err != nil {
		return nil, err
	}

	machineInfo := info.MachineInfo{}
	err = kip.doRequestAndGetValue(req, &machineInfo)
	if err != nil {
		return nil, err
	}

	return &machineInfo, nil
}

func newKubeletInfoProvider(client *kubelet.Client, logResponses bool) *kubeletInfoProvider {
	return &kubeletInfoProvider{
		client:       client,
		lastUpdate:   time.Now(),
		logResponses: logResponses,
	}
}

func (kip *kubeletInfoProvider) getAllContainers(start, end time.Time) ([]info.ContainerInfo, error) {
	// Request data from all subcontainers.
	request := statsRequest{
		ContainerName: "/",
		NumStats:      1,
		Start:         start,
		End:           end,
		Subcontainers: true,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	log.Debugf("Sending body to kubelet stats endpoint: %s", body)
	req, err := kip.client.NewRequest("POST", "/stats/container/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var containers map[string]info.ContainerInfo
	err = kip.doRequestAndGetValue(req, &containers)
	if err != nil {
		return nil, fmt.Errorf("failed to get all container stats from Kubelet URL %q: %v", req.URL.String(), err)
	}

	result := make([]info.ContainerInfo, 0, len(containers))
	for _, containerInfo := range containers {
		result = append(result, containerInfo)
	}
	return result, nil
}

func (kip *kubeletInfoProvider) doRequestAndGetValue(req *http.Request, value interface{}) error {
	response, err := kip.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read Kubelet response body - %v", err)
	}

	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Kubelet request resulted in 404: %s", req.URL.String())
	} else if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Kubelet request failed - %q, response: %q", response.Status, string(body))
	}

	kubeletAddr := "[unknown]"
	if req.URL != nil {
		kubeletAddr = req.URL.Host
	}
	if kip.logResponses {
		log.Debugf("Raw response from Kubelet at %s: %s", kubeletAddr, string(body))
	}

	err = json.Unmarshal(body, value)
	if err != nil {
		return fmt.Errorf("Failed to parse Kubelet output. Response: %q. Error: %v", string(body), err)
	}
	return nil
}
