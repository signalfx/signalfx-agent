package cluster

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

// KubernetesElector provides a simple way for monitors to only send
// metrics from a single instance of the agent in a Kubernetes cluster.  It
// wraps client-go's leaderelection tool, and uses the node name as the
// identifier in the election process, but this is scoped by namespace as well
// so there can be at most one agent pod per namespace per node for the logic
// to work. Calling this function starts the election process if it is not
// already running and returns a channel that gets fed true when this instance
// becomes leader and subsequently false if the instance stops being the leader
// for some reason, at which point the channel could send true again and so on.
// All monitors that need leader election will share the same election process.
// There is no way to stop the leader election process once it starts.
type KubernetesElector struct {
	v1Client corev1.CoreV1Interface
}

func NewKubernetesElector(v1Client corev1.CoreV1Interface) *KubernetesElector {
	return &KubernetesElector{
		v1Client: v1Client,
	}
}

var _ Elector = &KubernetesElector{}

func (kle *KubernetesElector) RunSelection(ctx context.Context, agentID string, key Key, callback ChangeCallback) error {
	ns := os.Getenv("MY_NAMESPACE")
	if ns == "" {
		return errors.New("MY_NAMESPACE envvar is not defined")
	}

	resLock, err := resourcelock.New(
		resourcelock.ConfigMapsResourceLock,
		ns,
		"signalfx-agent-"+key.Name+"-"+strconv.Itoa(int(key.Index)),
		kle.v1Client,
		resourcelock.ResourceLockConfig{
			Identity: agentID,
			// client-go can't make anything simple
			EventRecorder: &record.FakeRecorder{},
		})

	if err != nil {
		return err
	}

	le, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          resLock,
		LeaseDuration: 60 * time.Second,
		RenewDeadline: 45 * time.Second,
		RetryPeriod:   30 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ <-chan struct{}) {},
			OnStoppedLeading: func() {},
			OnNewLeader:      func(newLeaderID string) { callback(key, newLeaderID) },
		},
	})
	if err != nil {
		return err
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		le.Run()
	}
}
