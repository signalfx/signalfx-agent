# SignalFx Agent

[SignalFx](https://signalfx.com) is a cloud monitoring and alerting solution
for modern enterprise infrastructures.

## Introduction

This chart will deploy the SignalFx agent as a DaemonSet to all nodes in your
cluster.  It is designed to be run in only one release at a time.

See [the agent
docs](https://docs.signalfx.com/en/latest/integrations/kubernetes-quickstart.html)
for more information on how the agent works.  The installation steps will be
different since you are using Helm but the agent otherwise behaves identically.

## Use

To use this chart with Helm, add our SignalFx Helm chart repository to Helm
like this:

`$ helm repo add signalfx https://dl.signalfx.com/helm-repo`

Then to ensure the latest state of the repository, run:

`$ helm repo update`

Then you can install the agent using the chart name `signalfx/signalfx-agent`.

## Configuration

### Configuring your realm
By default, the Smart Agent will send data to the `us0` realm. If you are
not in this realm, you will need to explicitly set the `signalFxRealm` option
in the agent configuration. To determine if you are in a different realm,
check your profile page in the SignalFx web application.

### Values

See the [values.yaml](./values.yaml) file for more information on how to
configure releases.

There are two **required** config options to run this chart:
`signalFxAccessToken` and `clusterName` (if not overridding the agent config
template and providing your own cluster name).

It is also **recommended** that you explicitly specify `agentVersion` when
deploying a release so that the agent will not be unintentionally updated based
on updates of the helm chart from the repo.

We also highly recommend that you pin the Helm chart version that you are using
(with the `--version` flag to `helm install/upgrade`) so that you do not
receive inadvertant updates to resources that you don't want.

For example, a basic command line install setting these values would be:

`$ helm install --version <HELM CHART VERSION> --set signalFxAccessToken=<YOUR_ACCESS_TOKEN> --set clusterName=<YOUR_CLUSTER_NAME> --set agentVersion=<VERSION_NUMBER> --set signalFxRealm=<YOUR_SIGNALFX_REALM> signalfx/signalfx-agent`

If you want to provide your own agent configuration, you can do so with the
`agentConfig` value.  Otherwise, you can do a great deal of customization to
the provided config template using values.

If you are using OpenShift set `kubernetesDistro` to `openshift` to get
OpenShift-specific functionality:

`$ helm install --version <HELM_CHART_VERSION> --set signalFxAccessToken=<YOUR_ACCESS_TOKEN> --set clusterName=<YOUR_CLUSTER_NAME> --set agentVersion=<VERSION_NUMBER> --set signalFxRealm=<YOUR_SIGNALFX_REALM> signalfx/signalfx-agent --set kubernetesDistro=openshift`

### Windows

If you are deploying the agent to a mixed Linux/Windows cluster (as of the time
of writing, master components had to run on Linux nodes so a mixed cluster was
inevitable), you can use a Windows Docker container release of the agent.  
You must deploy two separate Helm releases, one for Linux and one
for Windows.  The Linux release will be normal -- you don't have to do anything
special in your Helm values to account for the additional Windows nodes. The
Helm chart has been updated to include a `nodeSelector` on the
`kubernetes.io/os` node label so that it will only deploy to Linux nodes by
default.

The Windows Helm release of the agent requires some special config to make work:

```yaml
# This triggers a set of tweaks to the deployment that make it work better for
# Windows.
isWindows: true
hostPath: C:\\hostfs
# Let the Linux Helm release do cluster metrics.
gatherClusterMetrics: false
# If your kube install is using CRI-O instead of docker, set the below to false.
gatherDockerMetrics: true

agentVersion: 5.20.1

# Kubelet on Windows doesn't seem to have the usage_bytes metrics so we'll
# transform the working set metric to it so that built-in content works.
kubeletMetricNameTransformations:
  container_memory_working_set_bytes: container_memory_usage_bytes
kubeletExtraMetrics:
  - container_memory_working_set_bytes

kubeletAPI:
  # We can't use hostNetworking on Windows so we have to connect to the node by
  # its IP address instead of localhost.
  url: 'https://${MY_NODE_IP}:10250'
  authType: serviceAccount
  skipVerify: true

image:
  repository: quay.io/signalfx/signalfx-agent
  # This is a special windows container release of the agent for Windows.
  tag: 5.20.1-windows
  pullPolicy: Always
```

You can deploy two separate Helm releases of the agent into the same namespace
on Kubernetes if you like, there will be no conflict.  Obviously the releases
just need to have separate names.
