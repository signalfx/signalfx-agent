package hostid

import "os"

// KubernetesNodeName returns the name of the current K8s node name, if running
// on K8s and if the appropriate envvar (MY_NODE_NAME) has been injected in the
// agent pod by the downward API mechanism.
func KubernetesNodeName() string {
	return os.Getenv("MY_NODE_NAME")
}
