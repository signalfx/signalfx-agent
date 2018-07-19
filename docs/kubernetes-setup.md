# Kubernetes Setup

The agent was first written for Kubernetes and is relatively easy to setup in a
cluster.  The agent is intended to be run on each node and will monitor
services running on those same nodes to minimize cross-node traffic.

See the documentation on [Monitoring Kubernetes](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information on how to use the UI components in the SignalFx webapp once
you are setup.

## Installation

Follow these instructions to install the SignalFx agent on your Kubernetes
cluster and configure it to auto-discover SignalFx-supported integrations to
monitor.

1. Store your organization's Access Token as a key named `access-token` in a
   Kubernetes secret named `signalfx-agent`:

   ```sh
    $ kubectl create secret generic --from-literal access-token=MY_ACCESS_TOKEN signalfx-agent
   ```

2. If you use [Helm](https://github.com/kubernetes/helm), you can use [our
   chart](https://github.com/kubernetes/charts/tree/master/stable/signalfx-agent)
   in the stable Helm chart repository.  Otherwise, download the following
   files from SignalFx's Github repository to the machine you usually run
   `kubectl` from, and modify them as indicated.

   -  [daemonset.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/k8s/daemonset.yaml):
       Kubernetes daemon set configuration
   -  [configmap.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/k8s/configmap.yaml):
       SignalFx agent configuration

     -  Using a text editor, replace the default value `MY-CLUSTER` with the
         desired name for your cluster. This will appear in the dimension
         called `kubernetes_cluster` in SignalFx.
     -  If the agent will be sending data via a proxy, see [proxy
         support](https://github.com/signalfx/signalfx-agent#proxy-support).
     -  If docker and cadvisor metrics are not necessary for certain
         containers, see [filtering](./filtering.md).

   -  If you have RBAC enabled in your cluster, you can look at the [other k8s resources](
       https://github.com/signalfx/signalfx-agent/tree/master/deployments/k8s)
       in the agent repo to see what is required for the agent pod to have the
       proper permissions.

      **If you are using Rancher for your Kubernetes deployment,** complete the
      instructions in [Rancher](#rancher) before proceeding with the next step.

	  **If you are using AWS Elastic Container Service for Kubernetes (EKS) for
	  your Kubernetes deployment,** complete the instructions in [AWS Elastic
	  Container Service for Kubernetes
	  (EKS)](#aws-elastic-container-service-for-kubernetes-eks) before
	  proceeding with the next step.

	  **If you are using Pivotal Container Service (PKS) for your Kubernetes
	  deployment,** complete the instructions in [Pivotal Container Service
	  (PKS)](#pivotal-container-service-pks) before proceeding with the next
	  step.

	  **If you are using Google Container Engine (GKE) for your Kubernetes
	  deployment,** complete the instructions in [Google Container Engine
	  (GKE)](#google-container-engine-gke) before proceeding with the next
	  step.

      **If you are using OpenShift 3.0+ for your Kubernetes deployment,** complete the
      instructions in [Openshift](#openshift) before proceeding with the next step.


3. Run the following commands on your Kubernetes cluster to install the agent
   with default configuration. Include the path to each .yaml file you
   downloaded in step #2.

   ```sh
      $ kubectl create -f configmap.yaml \
                       -f daemonset.yaml
   ```

4. Data will begin streaming into SignalFx. After a few minutes, verify that
   data from Kubernetes has arrived using the Infrastructure page.
   If you don't see data arriving, check the logs on a random agent container
   and see if there are any errors.  You can also exec the command
   `signalfx-agent status` in any of the agent pods to get a diagnostic output
   from the agent.


## Discovering your services

The SignalFx agent that is able to monitor Kubernetes environments is
pre-configured to include most of the integrations that SignalFx supports out
of the box. Using customizable rules that are based on the container image name
and service port, you can automatically start monitoring the microservices
running in the containers. Each integration has a default configuration that
you can customize for your environment by creating a new integration
configuration file.

For more information, see [Auto Discovery](./auto-discovery.md).

## Master nodes

Our provided agent DaemonSet includes a set of tolerations for master nodes
that should work across multiple K8s versions.  If your master node does not
use the taints included in the provided daemonset, you should replace the
tolerations with your cluster's master taint so that the agent will run on the
master node(s).

## Observers

Observers are what discover services running in the environment.  For
monitoring services, our agent is setup to monitor services running on the same
K8s node as the agent.

For Kubernetes, there are two observers that you can use:

 - [API Observer](./observers/k8s-api.md)
 - [Kubelet Observer](./observers/k8s-kubelet.md)

We recommend the API observer since the Kubelet API is technically undocumented.

## Monitors

Monitors are what collect metrics from the environment or services.  See
[Monitor Config](./monitor-config.md) for more information on specific monitors
that we support.  All of these work the same in Kubernetes.

Of particular relevance to Kubernetes are the following monitors:

 - [Kubernetes Cluster](./monitors/kubernetes-cluster.md) - Gets cluster level
	 metrics from the K8s API
 - [cAdvisor](./monitors/cadvisor.md) - Gets container metrics directly from
	 cAdvisor exposed on the same node (**most likely won't work in newer K8s
	 versions that don't expose cAdvisor's port in the Kubelet**)
 - [Kubelet Stats](./monitors/kubelet-stats.md) - Gets cAdvisor metrics through
	 the Kubelet `/stats` endpoint.  This is much more robust, as it uses the
	 same interface that Heapster uses.
 - [Prometheus Exporter](./monitors/prometheus-exporter.md) - Gets prometheus
	 metrics directly from exporters.  This is useful especially if you already
	 have exporters deployed in your cluster because you currently use
	 Prometheus.

If you want to pull metrics from
[kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) you can
use use a config similar to the following (assuming the kube-state-metrics
instance is running once in your cluster in a container using an image that has
the string "kube-state-metrics" in it):

```yaml
    - type: prometheus-exporter
      discoveryRule: container_image =~ "kube-state-metrics"
      disableHostDimensions: true
      disableEndpointDimensions: true
      extraDimensions:
        metric_source: kube-state-metrics
```

This uses the [prometheus-exporter](./monitors/prometheus-exporter.md) monitor
to pull metrics from the service.  It also disables a lot of host and endpoint
specific dimensions that are irrelevant to cluster-level metrics.  Note that
many of the metrics exposed by `kube-state-metrics` overlap with our own
[kubernetes-cluster](./monitors/kubernetes-cluster.md) monitor, so you probably
don't want to enable both unless you are using heavy filtering.

## Config via K8s annotations

When using the `k8s-api` observer, you can use Kubernetes pod annotations to
tell the agent how to monitor your services.  There are several annotations
that the `k8s-api` observer recognizes:

- `agent.signalfx.com/monitorType.<port>: "<monitor type>"` - Specifies the
	monitor type to use when monitoring the specified port.  If this value is
	present, any agent config will be ignored, so you must fully specify any
	non-default config values you want to use in annotations.  If this
	annotation is missing for a port but other config is present, you must have
	discovery rules or manually configured endpoints in your agent config to
	monitor this port; the other annotation config values will be merged into
	the agent config.

- `agent.signalfx.com/config.<port>.<configKey>: "<configValue>"` - Specifies
	a config option for the monitor that will monitor this endpoint.  The
	options are the same as specified in the monitor config reference.  Lists
	may be specified with the syntax `[a, b, c]` (YAML compact list) which
	will be deserialized to a list that will be provided to the monitor.
	Boolean values are the annotation string values `true` or
	`false`.  Integers can also be specified; they must be strings as the
	annotation value, but they will be interpreted as an integer if they don't
	contain any non-number characters.

- `agent.signalfx.com/configFromEnv.<port>.<configKey>: "<env var name>"` --
	Specifies a config option that will be pulled from an environment variable
	on the same container as the port being monitored.

- `agent.signalfx.com/configFromSecret.<port>.<configKey>:
	"<secretName>/<secretKey>"` -- Maps the value of a secret to a config
	option.  The `<secretKey>` is the key of the secret value within the
	`data` object of the actual K8s Secret resource.  Note that this requires
	the agent's service account to have the correct permissions to read the
	specified secret.

In all of the above, the `<port>` field can be either the port number of the
endpoint you want to monitor or the assigned name.  The config is specific to a
single port, which allows you to monitor multiple ports in a single pod and
container by just specifying annotations with different ports.

### Example

The following K8s pod spec and agent YAML configuration accomplish the same
thing:

K8s pod spec:

```yaml
    metadata:
      annotations:
        agent.signalfx.com/monitorType.jmx: "collectd/cassandra"
        agent.signalfx.com/config.jmx.intervalSeconds: "20"
        agent.signalfx.com/config.jmx.mBeansToCollect: "[cassandra-client-read-latency, threading]"
      labels:
        app: my-app
    spec:
      containers:
      - name: cassandra
        ports:
        - containerPort: 7199
          name: jmx
          protocol: TCP
       ......
```

Agent config:

```yaml
    monitors:
    - type: collectd/cassandra
      intervalSeconds: 20
      mBeansToCollect:
      - cassandra-client-read-latency
      - threading
```

If a pod has the `agent.signalfx.com/monitorType.*` annotation on it, that
pod will be excluded from the auto discovery mechanism and will be monitored
only with the given annotation configuration.  If you want to merge
configuration from the annotations with agent configuration, you must omit the
`monitorType` annotation and rely on auto discovery to find this endpoint.
At that time, config from both sources will be merged together, with pod
annotation config taking precedent.


## Rancher

If you are using Rancher to manage your Kubernetes cluster, perform these steps
after you complete step 3 in [Installation](#installation).

#### Using HTTP or HTTPS proxy

If the Rancher nodes are behind a proxy, ensure that the Docker engine has the
proxy configured so that it can pull the signalfx-agent Docker image from
quay.io. See the [Rancher
documentation](https://docs.rancher.com/os/networking/proxy-settings/) for
details on how to configure the proxy.

Use the following configuration for the cadvisor monitor:

```yaml
  monitors:
    - type: cadvisor
      cadvisorURL: http://localhost:9344
```

Cadvisor runs on port 9344 instead of the standard 4194.

When you have completed these steps, continue with step 3 in
[Installation](#installation).


## AWS Elastic Container Service for Kubernetes (EKS)
On EKS, machine ids are identical across worker nodes, which makes that value
useless for identification.  Therefore, there are two changes you should make
to the configmap to use the K8s node name instead of machine-id.

 1) In the configmap.yaml, change the top-level config option `sendMachineId`
    to `false`.  This will cause the agent to omit the machine_id dimension from
    all datapoints and instead send the `kubernetes_node` dimension on all
    datapoints emitted by the agent.

 2) Under the kubernetes-cluster monitor configuration, set the option
    `useNodeName: true`.  This will cause that monitor to sync node labels to the
    `kubernetes_node` dimension instead of the `machine_id` dimension.

Note that in EKS there is no concept of a "master" node (at least not that is
exposed via the K8s API) and so all nodes will be treated as workers.


## Pivotal Container Service (PKS)

See [AWS Elastic Container Service for
Kubernetes](#aws-elastic-container-service-for-kubernetes-eks) -- the setup for
PKS is identical because of the similar lack of reliable machine ids.


## Google Container Engine (GKE)

On GKE, access to the kubelet is highly restricted and service accounts will
not work (at least as of GKE 1.9.4).  In those environments, you can use the
alternative, non-secure port 10255 on the kubelet in the `kubelet-stats`
monitor to get container metrics.  The config for that monitor will look like:

```
monitors:
 - type: kubelet-stats
   kubeletAPI:
     authType: none
     url: http://localhost:10255
```

As long as you use our standard RBAC resources, this should be the only
modification needed to accommodate GKE.

## OpenShift

If you are using OpenShift 3.0+ for your Kubernetes deployment, perform these
steps after you complete step 2 in [Installation](#installation).

OpenShift 3.0 is based on Kubernetes and thus most of the above instructions
apply.  There are more restrictive security policies that disallow some of the
things our agent needs to be effective, such as running in privileged mode and
mounting host filesystems to the agent container, as well as reading from the
Kubelet and Kubernetes API with service accounts.

First we need a service account for the agent (you will need to be a cluster
administrator to do the following):

`oc create serviceaccount signalfx-agent`

We need to make this service account able to read information about the
cluster:

`oadm policy add-cluster-role-to-user cluster-reader
system:serviceaccount:default:signalfx-agent`

Next we need to add this service account to the privileged SCC.  Run `oc edit
scc privileged` and add the signalfx-agent service account at the end of the
users list:

```yaml
    users: ...
    - system:serviceaccount:default:signalfx-agent
```

Finally in the daemonset config for the agent, you need to add the name of the
service account created above.  Add the following line in the `spec` section of
the agent daemonset (see above for the base daemonset config file):

`serviceAccountName: signalfx-agent`

Now you should be able to follow the instructions above and have the agent
running in short order.
