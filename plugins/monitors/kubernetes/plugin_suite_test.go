package kubernetes

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestKubernetesMonitor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Monitor Suite")
}
