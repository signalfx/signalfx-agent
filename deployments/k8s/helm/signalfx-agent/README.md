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

See the [values.yaml](./values.yaml) file for more information on how to
configure releases.

There are two **required** config options to run this chart: `signalFxAccessToken`
and `clusterName` (if not overridding the agent config template and providing your own cluster name).

It is also **recommended** that you explicitly specify `agentVersion` when deploying a release so that the agent will not be unintentionally updated based on updates of the helm chart from the repo.

For example, a basic command line install setting these values would be:

`$ helm install --set signalFxAccessToken=<YOUR_ACCESS_TOKEN> --set clusterName=<YOUR_CLUSTER_NAME> --set agentVersion=<VERSION_NUMBER> --set signalFxRealm=<YOUR_SIGNALFX_REALM> signalfx/signalfx-agent`

If you want to provide your own agent configuration, you can do so with the
`agentConfig` value.  Otherwise, you can do a great deal of customization to
the provided config template using values.

If you are using OpenShift set `kubernetesDistro` to `openshift` to get
OpenShift-specific functionality:

`$ helm install --set signalFxAccessToken=<YOUR_ACCESS_TOKEN> --set clusterName=<YOUR_CLUSTER_NAME> --set agentVersion=<VERSION_NUMBER> --set signalFxRealm=<YOUR_SIGNALFX_REALM> signalfx/signalfx-agent --set kubernetesDistro=openshift`
