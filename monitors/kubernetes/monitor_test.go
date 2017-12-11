package kubernetes

import (
	"fmt"
	"net/url"
	"os"

	//"github.com/davecgh/go-spew/spew"

	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/watch"

	"github.com/signalfx/golib/datapoint"
	"github.com/signalfx/neo-agent/core/common/kubernetes"
	log "github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/signalfx/neo-agent/monitors/kubernetes/testhelpers"
)

var _ = Describe("Kubernetes plugin", func() {
	var config *Config
	var fakeK8s *FakeK8s
	var monitor *Monitor
	var dpChan chan *datapoint.Datapoint

	BeforeEach(func() {
		dpChan = make(chan *datapoint.Datapoint, 100)
		config = &Config{}
		config.IntervalSeconds = 1
		config.KubernetesAPI = &kubernetes.APIConfig{
			AuthType:   "none",
			SkipVerify: true,
		}

		fakeK8s = NewFakeK8s()
		fakeK8s.Start()
		K8sURL, _ := url.Parse(fakeK8s.URL())

		// The k8s go library picks these up -- they are set automatically in
		// containers running in a real k8s env
		os.Setenv("KUBERNETES_SERVICE_HOST", K8sURL.Hostname())
		os.Setenv("KUBERNETES_SERVICE_PORT", K8sURL.Port())

	})

	doSetup := func(alwaysClusterReporter bool, thisPodName string) {
		config.AlwaysClusterReporter = alwaysClusterReporter
		os.Setenv("MY_POD_NAME", thisPodName)

		os.Setenv("SFX_ACCESS_TOKEN", "deadbeef")

		monitor = &Monitor{}
		monitor.DPs = dpChan

		err := monitor.Configure(config)
		if err != nil {
			panic("K8s monitor config failed")
		}
	}

	AfterEach(func() {
		monitor.Shutdown()
	})

	// Making an int literal pointer requires a helper
	intp := func(n int32) *int32 { return &n }
	intValue := func(v datapoint.Value) int64 {
		return v.(datapoint.IntValue).Int()
	}

	waitForDatapoints := func(expected int) []*datapoint.Datapoint {
		dps := make([]*datapoint.Datapoint, 0)
		for {
			dp := &datapoint.Datapoint{}
			Eventually(dpChan, 2).Should(Receive(&dp))
			dps = append(dps, dp)
			if len(dps) >= expected {
				break
			}
		}
		return dps
	}

	expectIntMetric := func(dps []*datapoint.Datapoint, uidField, objUid string, metricName string, metricValue int) {
		matched := false
		for _, dp := range dps {
			dims := dp.Dimensions
			if dp.Metric == metricName && dims[uidField] == objUid {
				Expect(intValue(dp.Value)).To(Equal(int64(metricValue)), fmt.Sprintf("%s %s", objUid, metricName))
				matched = true
			}
		}
		Expect(matched).To(Equal(true), fmt.Sprintf("%s %s %d", objUid, metricName, metricValue))
	}

	expectIntMetricMissing := func(dps []*datapoint.Datapoint, uidField, objUid string, metricName string) {
		matched := false
		for _, dp := range dps {
			dims := dp.Dimensions
			if dp.Metric == metricName && dims[uidField] == objUid {
				matched = true
			}
		}
		Expect(matched).To(Equal(false), fmt.Sprintf("%s %s", objUid, metricName))
	}

	It("Sends pod phase metrics", func() {
		log.SetLevel(log.DebugLevel)
		fakeK8s.SetInitialList([]*v1.Pod{
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "test1",
					UID:  "abcd",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					ContainerStatuses: []v1.ContainerStatus{
						v1.ContainerStatus{
							Name:         "container1",
							RestartCount: 5,
						},
					},
				},
			},
		})

		doSetup(true, "")

		dps := waitForDatapoints(2)

		Expect(dps[0].Metric).To(Equal("kubernetes.pod_phase"))
		Expect(intValue(dps[0].Value)).To(Equal(int64(2)))
		Expect(dps[1].Metric).To(Equal("kubernetes.container_restart_count"))
		Expect(intValue(dps[1].Value)).To(Equal(int64(5)))

		dims := dps[0].Dimensions
		Expect(dims["metric_source"]).To(Equal("kubernetes"))

		fakeK8s.EventInput <- WatchEvent{watch.Added, &v1.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "pod2",
				UID:  "1234",
			},
			Status: v1.PodStatus{
				Phase: v1.PodFailed,
				ContainerStatuses: []v1.ContainerStatus{
					v1.ContainerStatus{
						Name:         "container2",
						RestartCount: 0,
					},
				},
			},
		}}

		dps = waitForDatapoints(4)
		expectIntMetric(dps, "kubernetes_pod_uid", "1234", "kubernetes.container_restart_count", 0)

		fakeK8s.EventInput <- WatchEvent{watch.Modified, &v1.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "pod2",
				UID:  "1234",
			},
			Status: v1.PodStatus{
				Phase: v1.PodFailed,
				ContainerStatuses: []v1.ContainerStatus{
					v1.ContainerStatus{
						Name:         "container2",
						RestartCount: 2,
					},
				},
			},
		}}

		dps = waitForDatapoints(4)
		expectIntMetric(dps, "kubernetes_pod_uid", "1234", "kubernetes.container_restart_count", 2)

		fakeK8s.EventInput <- WatchEvent{watch.Deleted, &v1.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "pod2",
				UID:  "1234",
			},
			Status: v1.PodStatus{
				Phase: v1.PodFailed,
				ContainerStatuses: []v1.ContainerStatus{
					v1.ContainerStatus{
						Name:         "container2",
						RestartCount: 2,
					},
				},
			},
		}}

		dps = waitForDatapoints(2)

		expectIntMetricMissing(dps, "kubernetes_pod_uid", "1234", "kubernetes.container_restart_count")

	}, 5)

	It("Sends Deployment metrics", func() {
		fakeK8s.SetInitialList([]*v1.Pod{
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "test1",
					UID:  "1234",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
		})

		fakeK8s.SetInitialList([]*v1beta1.Deployment{
			&v1beta1.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name: "deploy1",
					UID:  "abcd",
				},
				Spec: v1beta1.DeploymentSpec{
					Replicas: intp(int32(10)),
				},
				Status: v1beta1.DeploymentStatus{
					AvailableReplicas: 5,
				},
			},
			&v1beta1.Deployment{
				ObjectMeta: v1.ObjectMeta{
					Name: "deploy2",
					UID:  "efgh",
				},
				Spec: v1beta1.DeploymentSpec{
					Replicas: intp(int32(1)),
				},
				Status: v1beta1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
		})

		doSetup(true, "")

		dps := waitForDatapoints(6)

		By("Reporting on existing deployments")
		expectIntMetric(dps, "uid", "abcd", "kubernetes.deployment.desired", 10)
		expectIntMetric(dps, "uid", "abcd", "kubernetes.deployment.available", 5)
		expectIntMetric(dps, "uid", "efgh", "kubernetes.deployment.desired", 1)
		expectIntMetric(dps, "uid", "efgh", "kubernetes.deployment.available", 1)

		fakeK8s.EventInput <- WatchEvent{watch.Modified, &v1beta1.Deployment{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "extensions/v1beta1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "deploy2",
				UID:  "efgh",
			},
			Spec: v1beta1.DeploymentSpec{
				Replicas: intp(int32(1)),
			},
			Status: v1beta1.DeploymentStatus{
				AvailableReplicas: 0,
			},
		}}

		dps = waitForDatapoints(6)
		By("Responding to events pushed on the watch API")
		expectIntMetric(dps, "uid", "abcd", "kubernetes.deployment.desired", 10)
		expectIntMetric(dps, "uid", "abcd", "kubernetes.deployment.available", 5)
		expectIntMetric(dps, "uid", "efgh", "kubernetes.deployment.desired", 1)
		expectIntMetric(dps, "uid", "efgh", "kubernetes.deployment.available", 0)
	})

	Describe("Filtering", func() {
		BeforeEach(func() {
			fakeK8s.SetInitialList([]*v1.Pod{
				&v1.Pod{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test1",
						Namespace: "default",
						UID:       "abcd",
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:         "container1",
								RestartCount: 5,
							},
						},
					},
				},
			})

			fakeK8s.SetInitialList([]*v1beta1.Deployment{
				&v1beta1.Deployment{
					ObjectMeta: v1.ObjectMeta{
						Name: "deploy1",
						UID:  "abcd",
					},
					Spec: v1beta1.DeploymentSpec{
						Replicas: intp(int32(10)),
					},
					Status: v1beta1.DeploymentStatus{
						AvailableReplicas: 5,
					},
				},
			})
		})

	})

	It("Reports if first in pod list", func() {
		fakeK8s.SetInitialList([]*v1.Pod{
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-1",
					UID:  "abcd",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-2",
					UID:  "efgh",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
		})

		doSetup(false, "agent-1")

		dps := waitForDatapoints(2)
		Expect(dps).To(HaveLen(2))
	})

	It("Doesn't report if not first in pod list", func() {
		fakeK8s.SetInitialList([]*v1.Pod{
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-1",
					UID:  "abcd",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-2",
					UID:  "efgh",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
		})

		doSetup(false, "agent-2")

		Expect(dpChan).ShouldNot(Receive())
	})

	It("Starts reporting if later becomes first in pod list", func() {
		fakeK8s.SetInitialList([]*v1.Pod{
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-1",
					UID:  "abcd",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			&v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name: "agent-2",
					UID:  "efgh",
					Labels: map[string]string{
						"app": "signalfx-agent",
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
		})

		doSetup(false, "agent-2")

		Expect(dpChan).ShouldNot(Receive())

		fakeK8s.EventInput <- WatchEvent{watch.Deleted, &v1.Pod{
			TypeMeta: unversioned.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "agent-1",
				UID:  "abcd",
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		}}

		dps := waitForDatapoints(1)
		Expect(dps).To(HaveLen(1))
	})
})
