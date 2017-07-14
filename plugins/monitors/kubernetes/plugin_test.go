package kubernetes

import (
	"fmt"
	"log"
	"net/url"
	"os"

	//"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/watch"

	sfxproto "github.com/signalfx/com_signalfx_metrics_protobuf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/signalfx/neo-agent/plugins/monitors/kubernetes/testhelpers"
)

var _ = Describe("Kubernetes plugin", func() {
	var config *viper.Viper
	var fakeSignalFx *FakeSignalFx
	var fakeK8s *FakeK8s
	var plugin *Plugin

	BeforeEach(func() {
		config = viper.New()
		config.Set("intervalSeconds", 1)
		config.Set("authType", "none")
		config.Set("tls.skipVerify", true)
		config.Set("clusterName", "test-cluster")

		fakeSignalFx = NewFakeSignalFx()
		fakeSignalFx.Start()
		// This has to go on the global viper config until config is fixed
		viper.Set("ingesturl", fakeSignalFx.URL())

		fakeK8s = NewFakeK8s()
		fakeK8s.Start()
		K8sURL, _ := url.Parse(fakeK8s.URL())

		// The k8s go library picks these up -- they are set automatically in
		// containers running in a real k8s env
		os.Setenv("KUBERNETES_SERVICE_HOST", K8sURL.Hostname())
		os.Setenv("KUBERNETES_SERVICE_PORT", K8sURL.Port())

	})

	doSetup := func(alwaysClusterReporter bool, thisPodName string) {
		config.Set("alwaysClusterReporter", alwaysClusterReporter)
		os.Setenv("MY_POD_NAME", thisPodName)

		os.Setenv("SFX_ACCESS_TOKEN", "deadbeef")

		var err error
		plugin = plugins.MakePlugin(pluginType).(*Plugin)
		log.Printf("plugin: %p; %s", plugin, err)
		if err != nil {
			Fail(err.Error())
		}

		plugin.Configure(config)
	}

	AfterEach(func() {
		plugin.Shutdown()
	})

	// Making an int literal pointer requires a helper
	intp := func(n int32) *int32 { return &n }

	expectIntMetric := func(dps []*sfxproto.DataPoint, uidField, objUid string, metricName string, metricValue int) {
		matched := false
		for _, dp := range dps {
			dims := ProtoDimensionsToMap(dp.GetDimensions())
			if dp.GetMetric() == metricName && dims[uidField] == objUid {
				Expect(dp.GetValue().GetIntValue()).To(Equal(int64(metricValue)))
				matched = true
			}
		}
		Expect(matched).To(Equal(true), fmt.Sprintf("%s %s %d", objUid, metricName, metricValue))
	}

	It("Sends pod phase metrics", func() {
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

		dps := fakeSignalFx.PopIngestedDatapoints()

		Expect(dps).To(HaveLen(2))

		Expect(dps[0].GetMetric()).To(Equal("kubernetes.pod_phase"))
		Expect(dps[0].GetValue().GetIntValue()).To(Equal(int64(2)))
		Expect(dps[1].GetMetric()).To(Equal("kubernetes.container_restart_count"))
		Expect(dps[1].GetValue().GetIntValue()).To(Equal(int64(5)))

		dims := ProtoDimensionsToMap(dps[0].GetDimensions())
		Expect(dims["metric_source"]).To(Equal("kubernetes"))
		Expect(dims["kubernetes_cluster"]).To(Equal("test-cluster"))

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

		dps = fakeSignalFx.PopIngestedDatapoints()
		Expect(dps).To(HaveLen(4))
		expectIntMetric(dps, "pod_uid", "1234", "kubernetes.container_restart_count", 0)

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

		dps = fakeSignalFx.PopIngestedDatapoints()
		Expect(dps).To(HaveLen(4))
		expectIntMetric(dps, "pod_uid", "1234", "kubernetes.container_restart_count", 2)

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

		dps = fakeSignalFx.PopIngestedDatapoints()
		Expect(dps).To(HaveLen(2))

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

		var dps []*sfxproto.DataPoint
		Eventually(func() int {
			dps = fakeSignalFx.PopIngestedDatapoints()
			return len(dps)
		}, 5).Should(BeNumerically(">", 2))

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

		dps = fakeSignalFx.PopIngestedDatapoints()
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

		It("Filters out excluded metrics", func() {
			config.Set("metricFilter", []string{"!kubernetes.pod_phase"})
			doSetup(true, "")

			dps := fakeSignalFx.PopIngestedDatapoints()

			metrics := make([]string, 0, len(dps))
			for _, dp := range dps {
				metrics = append(metrics, dp.GetMetric())
			}
			Expect(metrics).To(Not(ContainElement("kubernetes.pod_phase")))
			Expect(metrics).To(ContainElement("kubernetes.container_restart_count"))
		})

		It("Filters out non-included metrics", func() {
			config.Set("metricFilter", []string{"kubernetes.pod_phase"})
			doSetup(true, "")

			dps := fakeSignalFx.PopIngestedDatapoints()

			metrics := make([]string, 0, len(dps))
			for _, dp := range dps {
				metrics = append(metrics, dp.GetMetric())
			}
			Expect(metrics).To(ContainElement("kubernetes.pod_phase"))
			Expect(metrics).To(Not(ContainElement("kubernetes.container_restart_count")))
		})

		It("Filters out excluded namespaces", func() {
			config.Set("namespaceFilter", []string{"!default"})
			doSetup(true, "")

			dps := fakeSignalFx.PopIngestedDatapoints()

			metrics := make([]string, 0, len(dps))
			for _, dp := range dps {
				metrics = append(metrics, dp.GetMetric())
			}
			Expect(metrics).To(Not(ContainElement("kubernetes.pod_phase")))
			Expect(metrics).To(ContainElement("kubernetes.deployment.desired"))
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

		dps := fakeSignalFx.PopIngestedDatapoints()

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

		fakeSignalFx.EnsureNoDatapoints()
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

		fakeSignalFx.EnsureNoDatapoints()

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

		dps := fakeSignalFx.PopIngestedDatapoints()
		Expect(dps).To(HaveLen(1))
	})
})
