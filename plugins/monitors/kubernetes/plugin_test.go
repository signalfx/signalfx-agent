package kubernetes

import (
	"log"
	"net/url"
	"os"

	//"github.com/davecgh/go-spew/spew"
	"github.com/signalfx/neo-agent/plugins"
	"github.com/spf13/viper"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/signalfx/neo-agent/plugins/monitors/kubernetes/testhelpers"
)

var _ = Describe("Kubernetes Monitor", func() {
	var fakeSignalFx *FakeSignalFx
	var fakeK8s *FakeK8s
	var monitor plugins.IPlugin

	doSetup := func(alwaysReport bool, thisPodName string) {
		config := viper.New()
		config.Set("intervalSeconds", 1)
		config.Set("alwaysReport", alwaysReport)
		os.Setenv("MY_POD_NAME", thisPodName)
		config.Set("authType", "none")
		config.Set("tls.skipVerify", true)

		fakeSignalFx = NewFakeSignalFx()
		fakeSignalFx.Start()
		config.Set("ingesturl", fakeSignalFx.URL())

		fakeK8s = NewFakeK8s()
		fakeK8s.Start()
		k8sUrl, _ := url.Parse(fakeK8s.URL())

		// The k8s go library picks these up -- they are set automatically in
		// containers running in a real k8s env
		os.Setenv("KUBERNETES_SERVICE_HOST", k8sUrl.Hostname())
		os.Setenv("KUBERNETES_SERVICE_PORT", k8sUrl.Port())
		os.Setenv("SFX_ACCESS_TOKEN", "deadbeef")

		var err error
		monitor, err = NewKubernetesMonitorPlugin("monitors/kubernetes", config)
		log.Printf("monitor: %p; %s", monitor, err)
		if err != nil {
			Fail(err.Error())
		}
	}

	AfterEach(func() {
		monitor.Stop()
	})

	It("Sends pod phase metrics", func() {
		defer GinkgoRecover()

		doSetup(true, "")

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

		monitor.Start()

		dps := fakeSignalFx.GetIngestedDatapoints()

		Expect(dps).To(HaveLen(2))

		Expect(dps[0].GetMetric()).To(Equal("kubernetes.pod_phase"))
		Expect(dps[0].GetValue().GetIntValue()).To(Equal(int64(2)))
		Expect(dps[1].GetMetric()).To(Equal("kubernetes.container_restart_count"))
		Expect(dps[1].GetValue().GetIntValue()).To(Equal(int64(5)))

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

		dps = fakeSignalFx.GetIngestedDatapoints()
		Expect(dps).To(HaveLen(4))
		matched := false
		for _, dp := range dps {
			dims := ProtoDimensionsToMap(dp.GetDimensions())
			if dp.GetMetric() == "kubernetes.container_restart_count" && dims["pod_uid"] == "1234" {
				Expect(dp.GetValue().GetIntValue()).To(Equal(int64(0)))
				matched = true
			}
		}
		Expect(matched).To(Equal(true))

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

		dps = fakeSignalFx.GetIngestedDatapoints()
		Expect(dps).To(HaveLen(4))
		matched = false
		for _, dp := range dps {
			dims := ProtoDimensionsToMap(dp.GetDimensions())
			if dp.GetMetric() == "kubernetes.container_restart_count" && dims["pod_uid"] == "1234" {
				Expect(dp.GetValue().GetIntValue()).To(Equal(int64(2)))
				matched = true
			}
		}
		Expect(matched).To(Equal(true))

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

		dps = fakeSignalFx.GetIngestedDatapoints()
		Expect(dps).To(HaveLen(2))

	}, 5)

	//It("Sends Deployment metrics", func() {
	//})

	//It("Filters out unwanted namespaces and metrics", func() {
	//})

	It("Reports if first in pod list", func() {
		doSetup(false, "agent-1")
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

		monitor.Start()

		dps := fakeSignalFx.GetIngestedDatapoints()

		Expect(dps).To(HaveLen(2))
	})

	It("Doesn't report if not first in pod list", func() {
		doSetup(false, "agent-2")
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

		monitor.Start()

		fakeSignalFx.EnsureNoDatapoints()
	})
})
