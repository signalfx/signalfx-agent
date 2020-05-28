# Kubernetes Deployments

The agent runs in Kubernetes and has some monitors and observers specific to
that environment.

The resources in this directory can be used to deploy the agent to K8s.  They
are generated from our [Helm](https://github.com/kubernetes/helm) chart, which
is available in a SignalFx Helm repository -- see [SignalFx Agent Helm Chart
Use](./helm/signalfx-agent#use) for more information.

A few things to do before deploying these directly (when **not using Helm**):

 1. Make sure you change the `kubernetes_cluster` global dimension *and* the
	`cluster` config option to something specific to your cluster in the
	`configmap.yaml` resource before deploying.

 2. Also make sure you change the `namespace` of the service account token
	reference in [./clusterrolebinding.yaml](./clusterrolebinding.yaml) to the
	namespace in which you are deploying the agent.

 3. Create a secret in K8s with your org's access token:

	`kubectl create secret generic --from-literal access-token=MY_ACCESS_TOKEN signalfx-agent`

Then to deploy run the following from the present directory:

`cat *.yaml | kubectl apply -f -`

## Host Networking

The agent runs with host networking by default (the `hostNetwork: true` option
in the DaemonSet).  We intent to move away from that at some point, but if you
want to go ahead and stop using host networking for some reason (e.g. the agent
has trouble addressing service pods or there are DNS resolution issues), you
can make the agent run with its own network namespace by doing the following:

 1. Change `hostNetwork: true` to `hostNetwork: false` in the DaemonSet.
 2. Remove the `dnsPolicy` setting or change it to `dnsPolicy: ClusterFirst`.
 3. Add the item `hostname: ${MY_NODE_NAME}` under `agent.yaml` in the agent
	ConfigMap.
 4. Configure the `kubelet-metrics` monitor to use the node name as the hostname
	by using the following config:

	```
    - type: kubelet-metrics
      kubeletAPI:
        url: https://${MY_NODE_NAME}:10250
        authType: serviceAccount
    ```

	If you have a non-standard `kubelet-metrics` config, alter this accordingly.
	Note that this **requires that node names are valid DNS hosts as well** and
	it will not work if node names are not resolvable.  Of course, cluster
	firewalls also have to allow for traffic from the pod network to the
	kubelets.

This requires version 3.6.2 or later of the agent to work.

## AWS EKS/Fargate
If you are running on AWS EKS with Fargate profiles, you will need to deploy a
special instance of the agent within the cluster in such a way that it can
access the Fargate virtual nodes and pods.  The simplest way to do this is to
just run the agent as a single-replica Deployment in a namespace that a Fargate
profile covers.  The signalfx-agent Helm chart supports this by specifying the
chart value `isServerless: true`.  This will skip the creation of the normal
DaemonSet in favor of a 1-replica Deployment.  There will also be special
configuration to automatically discover all pods on all nodes within the
cluster, as well as config to make the agent discover and scrape container
metrics from the virtual kubelets that are created by EKS.  All traditional
host infrastructure monitors will be disabled if `isServerless: true`.

Two notable limitations when running the agent as a Deployment in Fargate:

 - Due to network configuration set by AWS, the agent pod cannot access its own
   kubelet, which means container stats (those from `kubelet-metrics`) will not
   be emitted for the agent container.

 - All pod monitoring must be done in a single, centralized agent instance.
   The agent should scale vertically quite well so more CPU or (to a lesser
   extent) memory will help it monitor a very large Fargate-based cluster if
   needed.  You can also deploy the agent as a sidecar container on each pod
   but this is generally unnecessary and somewhat wasteful of resources.

The [serverless](./serverless) directory contains a set of sample YAML
resources that you can start from to deploy the agent in a serverless K8s
environment.

## Development

These resources can be refreshed from the Helm chart by using the
`generate-from-helm` script in this dir.
