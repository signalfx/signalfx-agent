package poller

import (
	"fmt"
	"reflect"
	"regexp"
	//"sort"

	"sync"
	"time"

	"log"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/monitors/cadvisor/converter"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api/v1"
	kubeFields "k8s.io/client-go/pkg/fields"
	kubeLabels "k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/rest"

	"github.com/google/cadvisor/client"
	info "github.com/google/cadvisor/info/v1"
)

func init() {
	re = regexp.MustCompile(`^k8s_(?P<container_name>[^_]+)_(?P<kubernetes_pod_name>[^_]+)_(?P<kubernetes_namespace>[^_]+)`)
	reCaptureNames = re.SubexpNames()
}

var re *regexp.Regexp
var reCaptureNames []string

// ToolVersion set by build system
var ToolVersion = "NOT SET"

// Config for prometheusScraper
type Config struct {
	IngestURL              string
	CadvisorURL            []string
	APIToken               string
	DataSendRate           string
	ClusterName            string
	NodeServiceRefreshRate string
	CadvisorPort           int
	KubernetesURL          string
	KubernetesUsername     string
	KubernetesPassword     string
	Hostname               string
	DefaultDimensions      map[string]string
	ExcludedNames          []*regexp.Regexp
	ExcludedImages         []*regexp.Regexp
	ExcludedLabels         [][]*regexp.Regexp
	ExcludedMetrics        map[string]bool
}

// workProxy will call work.DoWork and then callback
/*type workProxy struct {
	work     workpool.PoolWorker
	callback func()
}

func (wp *workProxy) DoWork(workRoutine int) {
	wp.work.DoWork(workRoutine)
	wp.callback()
}*/

type scrapWork2 struct {
	serverURL  string
	collector  *converter.CadvisorCollector
	chRecvOnly chan datapoint.Datapoint
}

func (scrapWork *scrapWork2) DoWork(workRoutine int) {
	scrapWork.collector.Collect(scrapWork.chRecvOnly)
}

/*type sortableDatapoint []*datapoint.Datapoint

func (sd sortableDatapoint) Len() int {
	return len(sd)
}

func (sd sortableDatapoint) Swap(i, j int) {
	sd[i], sd[j] = sd[j], sd[i]
}

func (sd sortableDatapoint) Less(i, j int) bool {
	return sd[i].Timestamp.Unix() < sd[j].Timestamp.Unix()
}*/

type cadvisorInfoProvider struct {
	cc         *client.Client
	lastUpdate time.Time
}

func (cip *cadvisorInfoProvider) SubcontainersInfo(containerName string) ([]info.ContainerInfo, error) {
	curTime := time.Now()
	info, err := cip.cc.AllDockerContainers(&info.ContainerInfoRequest{Start: cip.lastUpdate, End: curTime})
	if len(info) > 0 {
		cip.lastUpdate = curTime
	}
	return info, err
}

func (cip *cadvisorInfoProvider) GetMachineInfo() (*info.MachineInfo, error) {
	return cip.cc.MachineInfo()
}

func newCadvisorInfoProvider(cadvisorClient *client.Client) *cadvisorInfoProvider {
	return &cadvisorInfoProvider{
		cc:         cadvisorClient,
		lastUpdate: time.Now(),
	}
}

const autoFlushTimerDuration = 500 * time.Millisecond
const maxDatapoints = 50
const dataSourceType = "kubernetes"

func printVersion() {
	log.Printf("git build commit: %v\n", ToolVersion)
}

func nameToLabel(name string) map[string]string {
	extraLabels := map[string]string{}
	matches := re.FindStringSubmatch(name)
	for i, match := range matches {
		if len(reCaptureNames[i]) > 0 {
			extraLabels[re.SubexpNames()[i]] = match
		}
	}
	return extraLabels
}

func updateNodes(kubeClient *corev1.CoreV1Client, cPort int) (hostIPtoNodeMap map[string]v1.Node, nodeIPs []string) {
	hostIPtoNodeMap = make(map[string]v1.Node, 2)
	nodeIPs = make([]string, 0, 2)
	nodeList, apiErr := kubeClient.Nodes().List(v1.ListOptions{
		LabelSelector: kubeLabels.Everything().String(),
		FieldSelector: kubeFields.Everything().String(),
	})
	if apiErr != nil {
		log.Printf("Failed to list kubernetes nodes. Error: %v\n", apiErr)
	} else {
		for _, node := range nodeList.Items {
			var hostIP string
			for _, nodeAddress := range node.Status.Addresses {
				switch nodeAddress.Type {
				case v1.NodeInternalIP:
					hostIP = nodeAddress.Address
					break
				case v1.NodeLegacyHostIP:
					hostIP = nodeAddress.Address
				}
			}
			if hostIP != "" {
				hostIP = fmt.Sprintf("http://%v:%v", hostIP, cPort)
				nodeIPs = append(nodeIPs, hostIP)
				hostIPtoNodeMap[hostIP] = node
			}
		}
	}

	return hostIPtoNodeMap, nodeIPs
}

func updateServices(kubeClient *corev1.CoreV1Client) (podToServiceMap map[string]string) {

	serviceList, apiErr := kubeClient.Services("").List(v1.ListOptions{
		LabelSelector: kubeLabels.Everything().String(),
		FieldSelector: kubeFields.Everything().String(),
	})
	if apiErr != nil {
		log.Printf("Failed to list kubernetes services. Error: %v\n", apiErr)
		return nil
	}

	podToServiceMap = make(map[string]string, 2)
	for _, service := range serviceList.Items {
		podList, apiErr := kubeClient.Pods("").List(v1.ListOptions{
			LabelSelector: kubeLabels.SelectorFromSet(service.Spec.Selector).String(),
			FieldSelector: kubeFields.Everything().String(),
		})
		if apiErr != nil {
			log.Printf("Failed to list kubernetes pods. Error: %v\n", apiErr)
		} else {
			for _, pod := range podList.Items {
				//fmt.Printf("%v -> %v\n", pod.ObjectMeta.Name, service.ObjectMeta.Name)
				podToServiceMap[pod.ObjectMeta.Name] = service.ObjectMeta.Name
			}
		}
	}
	return podToServiceMap
}

func newKubeClient(config *Config) (restClient *corev1.CoreV1Client, err error) {
	var kubeConfig *rest.Config
	if config.KubernetesURL == "" {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("Failed to create kubernetes client config. Error: %v\n", err)
		}
	} else {
		kubeConfig = &rest.Config{
			Host:     config.KubernetesURL,
			Username: config.KubernetesUsername,
			Password: config.KubernetesPassword,
			Insecure: true,
		}
	}

	restClient, err = corev1.NewForConfig(kubeConfig)

	if err != nil {
		log.Printf("Failed to create kubernetes client. Error: %v\n", err)
		return nil, err
	}

	return restClient, err
}

// MonitorNode collects metrics from a single node
func MonitorNode(cfg *Config, dpChan chan<- *datapoint.Datapoint, dataSendRate time.Duration) (stop chan bool, stopped chan bool, err error) {
	swc := newScrapWorkCache(cfg, dpChan)
	cadvisorClient, err := client.NewClient(cfg.CadvisorURL[0])
	if err != nil {
		return nil, nil, err
	}

	collector := converter.NewCadvisorCollector(newCadvisorInfoProvider(cadvisorClient), nameToLabel, cfg.ExcludedImages, cfg.ExcludedNames, cfg.ExcludedLabels, cfg.ExcludedMetrics)

	// TODO: fill in if we want node dimensions but that requires contacting apiserver.
	// swc.hostIPtoNameMap[]

	sw2 := &scrapWork2{
		// I think only used for swc.HostIPToNameMap lookup
		serverURL:  "",
		collector:  collector,
		chRecvOnly: make(chan datapoint.Datapoint),
	}

	swc.addWork(sw2)

	ticker := time.NewTicker(dataSendRate)
	stop = make(chan bool, 1)
	stopped = make(chan bool, 1)

	go func() {
		for {
			select {
			case <-stop:
				log.Println("stopping collection")
				ticker.Stop()
				close(sw2.chRecvOnly)
				return
			case <-ticker.C:
				collector.Collect(sw2.chRecvOnly)
			}
		}
	}()

	go func() {
		swc.waitAndForward()
		stopped <- true
		log.Println("waitAndForward returned")
	}()

	return stop, stopped, nil
}

type responseChannel *chan bool

type scrapWorkCache struct {
	workCache       []*scrapWork2
	cases           []reflect.SelectCase
	flushChan       chan responseChannel
	podToServiceMap map[string]string
	hostIPtoNameMap map[string]v1.Node
	dpChan          chan<- *datapoint.Datapoint
	cfg             *Config
	mutex           *sync.Mutex
}

func newScrapWorkCache(cfg *Config, dpChan chan<- *datapoint.Datapoint) *scrapWorkCache {
	return &scrapWorkCache{
		workCache: make([]*scrapWork2, 0, 1),
		cases:     make([]reflect.SelectCase, 0, 1),
		flushChan: make(chan responseChannel, 1),
		dpChan:    dpChan,
		cfg:       cfg,
		mutex:     &sync.Mutex{},
	}
}

func (swc *scrapWorkCache) addWork(work *scrapWork2) {
	swc.mutex.Lock()
	defer swc.mutex.Unlock()

	swc.workCache = append(swc.workCache, work)
	c := reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(work.chRecvOnly)}
	swc.cases = append(swc.cases, c)
}

// Build list of work
func (swc *scrapWorkCache) buildWorkList(URLList []string) {
	for _, serverURL := range URLList {
		cadvisorClient, localERR := client.NewClient(serverURL)
		if localERR != nil {
			log.Printf("Failed connect to server: %v\n", localERR)
			continue
		}

		swc.addWork(&scrapWork2{
			serverURL:  serverURL,
			collector:  converter.NewCadvisorCollector(newCadvisorInfoProvider(cadvisorClient), nameToLabel, []*regexp.Regexp{}, []*regexp.Regexp{}, [][]*regexp.Regexp{}, map[string]bool{}),
			chRecvOnly: make(chan datapoint.Datapoint),
		})
	}
}

func (swc *scrapWorkCache) setPodToServiceMap(m map[string]string) {
	swc.mutex.Lock()
	defer swc.mutex.Unlock()

	swc.podToServiceMap = m
}

func (swc *scrapWorkCache) setHostIPtoNameMap(m map[string]v1.Node) {
	swc.mutex.Lock()
	defer swc.mutex.Unlock()

	swc.hostIPtoNameMap = m
}

type eachWorkFunc func(int, *scrapWork2) bool

// foreachWork iterates over scrapWorkCache.workCache and calls eachWorkFunc on every element
// foreachWork will operate on copy of scrapWorkCache.workCache
func (swc *scrapWorkCache) foreachWork(f eachWorkFunc) {
	swc.mutex.Lock()
	workCacheCopy := make([]*scrapWork2, len(swc.workCache))
	copy(workCacheCopy, swc.workCache)
	swc.mutex.Unlock()

	for index, work := range workCacheCopy {
		if !f(index, work) {
			return
		}
	}
}

// This function will block
func (swc *scrapWorkCache) flush() {
	respChan := make(chan bool, 1)
	swc.flushChan <- &respChan
	<-respChan
}

func (swc *scrapWorkCache) fillNodeDims(chosen int, dims map[string]string) {

	node, ok := func() (n v1.Node, b bool) {
		swc.mutex.Lock()
		defer func() {
			swc.mutex.Unlock()
			if r := recover(); r != nil {
				log.Println("Recovered in fillNodeDims: ", r)
			}
		}()

		n, b = swc.hostIPtoNameMap[swc.workCache[chosen].serverURL]
		return
	}()

	if ok {
		dims["node"] = node.ObjectMeta.Name
		dims["node_container_runtime_version"] = node.Status.NodeInfo.ContainerRuntimeVersion
		dims["node_kernel_version"] = node.Status.NodeInfo.KernelVersion
		dims["node_kubelet_version"] = node.Status.NodeInfo.KubeletVersion
		dims["node_os_image"] = node.Status.NodeInfo.OSImage
		dims["node_kubeproxy_version"] = node.Status.NodeInfo.KubeProxyVersion
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// Wait on channel input and forward datapoints to SignalFx.
// This function will block.
func (swc *scrapWorkCache) waitAndForward() {
	swc.mutex.Lock()
	remaining := len(swc.cases)
	swc.mutex.Unlock()

	// localMutex used to sync i access
	localMutex := &sync.Mutex{}
	i := 0

	// ret is buffer that accumulates datapoints to be send to SignalFx
	ret := make([]*datapoint.Datapoint, maxDatapoints)

	autoFlushTimer := time.NewTimer(autoFlushTimerDuration)
	stopFlusher := make(chan bool, 1)
	flushFunc := func(respChan responseChannel) {
		func() {
			localMutex.Lock()
			defer localMutex.Unlock()

			if i > 0 {
				min := min(i, maxDatapoints)
				for i := 0; i < min; i++ {
					swc.dpChan <- ret[i]
				}
				i = 0
			}
		}()

		if respChan != nil {
			*respChan <- true
		}
	}

	resetMutex := &sync.Mutex{}
	resetFlushTimer := func() {
		resetMutex.Lock()
		defer resetMutex.Unlock()
		autoFlushTimer.Reset(autoFlushTimerDuration)
	}

	// This thread will flush ret buffer if requested
	// Also it will auto flush it in 500 milliseconds
	go func() {
		for true {
			select {
			case respChan := <-swc.flushChan:
				flushFunc(respChan)
			case <-autoFlushTimer.C:
				flushFunc(nil)
				resetFlushTimer()
			case <-stopFlusher:
				return
			}
		}
	}()

	for remaining > 0 {
		resetFlushTimer()
		chosen, value, ok := reflect.Select(swc.cases)
		autoFlushTimer.Stop()
		if !ok {
			// The chosen channel has been closed, so remove the case and work
			swc.mutex.Lock()
			swc.cases[chosen].Chan = reflect.ValueOf(nil)
			swc.cases = append(swc.cases[:chosen], swc.cases[chosen+1:]...)
			swc.workCache = append(swc.workCache[:chosen], swc.workCache[chosen+1:]...)
			remaining = len(swc.cases)
			swc.mutex.Unlock()
			continue
		}

		dp := value.Interface().(datapoint.Datapoint)
		dims := dp.Dimensions

		// filter POD level metrics
		if dims["container_name"] == "POD" {
			matched, _ := regexp.MatchString("^pod_network_.*", dp.Metric)
			if !matched {
				continue
			}
			delete(dims, "container_name")
		}

		dims["metric_source"] = dataSourceType
		dims["host"] = swc.cfg.Hostname

		swc.fillNodeDims(chosen, dims)

		for k, v := range swc.cfg.DefaultDimensions {
			dims[k] = v
		}

		// remove high cardinality dimensions
		delete(dims, "id")
		delete(dims, "name")

		func() {
			localMutex.Lock()
			defer localMutex.Unlock()

			ret[i] = &dp
			i++
			if i == maxDatapoints {
				//sort.Sort(sortableDatapoint(ret))

				func() {
					localMutex.Unlock()
					defer localMutex.Lock()

					swc.flush()
				}()
			}
		}()
	}
	stopFlusher <- true
}
