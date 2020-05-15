# Install Using kubectl

## Prerequisites

* Tasks

  Remove collector services such as `collectd`

  Remove third-party instrumentation and agent software
  **Note:**

  Do not use automatic instrumentation or instrumentation agents from
  other vendors when you're using SignalFx instrumentation. The results
  are unpredictable, but your instrumentation may break and your
  application may crash.

* Linux kernel version 2.6 or higher
* `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
* `terminal` or a similar command line interface application
* Org token (access token) from your SignalFx organization
* SignalFx realm from your SignalFx organization

## Configuration

To install the Smart Agent using kubectl, you need to create a
Kubernetes **secret** for your org token and update settings in the Agent's configuration files.

1. To create the Kubernetes secret, log in to the host from which you run kubectl.
2. Run the following command to create the Kubernetes secret `signalfx-agent`, substituting your org token for `<org_token>`:
`kubectl create secret generic --from-literal access-token=<org_token> signalfx-agent`
3. On the same host, download the following configuration files from the
[Kubernetes Deployments](https://github.com/signalfx/signalfx-agent/tree/master/deployments/k8s) area in GitHub:

| File                      | Description                                                                |
|:--------------------------|:---------------------------------------------------------------------------|
| `clusterrole.yaml`        | Configuration settings for the ClusterRole cluster-admin¹                  |
| `clusterrolebinding.yaml` | Configuration settings for Kubernetes role-based access control (**RBAC**) |
| `configmap.yaml`          | Cluster configuration settings                                             |
| `daemonset.yaml`          | Daemonset configuration settings                                           |
| `serviceaccount.yaml`     | Configuration settings for Kubernetes Service Accounts¹                    |
¹These configuration files don't require any updates.

4. Update `configmap.yaml`:
- **Cluster name:** For each of your Kubernetes clusters, replace `MY-CLUSTER` with a unique cluster name.
- **Realm:** Update the value of `signalFxRealm` with the name of your SignalFx realm.
- To avoid sending docker and cadvisor metrics being sent from some containers,
update the `datapointsToExclude` property. To learn more, see [Filtering](https://docs.signalfx.com/en/latest/integrations/agent/filtering.html#filtering).
5. Update `clusterrolebinding.yaml`:
- Update `MY_AGENT_NAMESPACE` or the service account token reference with the Smart Agent namespace in which you're deploying the agent.
6. Update `daemonset.yaml`:
- For RBAC-enabled clusters, add the permissions required for the Smart Agent.
- For **Rancher** nodes, ensure that the Docker engine proxy is configured so that it can pull the `signalfx-agent` Docker image from `quay.io`.
To learn more, see the Rancher v1.6 or Rancher v2.x documentation regarding proxy configuration.
- Update the **cAdvisor** monitor configuration to use port 9344:

```
monitors:
- type: cadvisor
 cadvisorURL: http://localhost:9344
```

- **If you're using OpenShift:**
- If you can't use the default namespace, modify each namespace occurrence and then ask your cluster administrator to run the following commands:

```
oc create serviceaccount signalfx-agent
oc adm policy add-cluster-role-to-user anyuid system:serviceaccount:default:signalfx-agent
oc edit scc privileged
users: ...
- system:serviceaccount:default:signalfx-agent

serviceAccountName: signalfx-agent
```

## Installation

Run the following command to update `kubetctl` with the configuration files you've just modified:
`cat *.yaml | kubectl apply -f-`

## Verify the Smart Agent

After you install the Smart Agent, it starts sending data from your clusters to SignalFx.

To see the services the Smart Agent has discovered, run the following command inside any of your Smart Agent containers:

```
while read -r line; do kubectl exec --namespace `echo $line` signalfx-agent status; done <<< `kubectl get pods -l app=signalfx-agent --all-namespaces --no-headers | tr -s " " | cut -d " " -f 1,2`
```

In addition, you can do the following in the SignalFx UI:

* For infrastructure monitoring:
  - In SignalFx UI, open the **Infrastructure** built-in dashboard
  - In the override bar at the top of the back, select **Choose a host**. Select one of your nodes from the dropdown.
  - The charts display metrics from the infrastructure for that node.
 To learn more, see [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

* For Kubernetes monitoring:
  - In SignalFx UI, from the main menu select **Infrastructure** > **Kubernetes Navigator** > **Cluster map**.
  - The map displays all the clusters running the Smart Agent
  - Click the magnification icon to view the nodes in a cluster.
  - The detail pane on the right hand side of the page displays details of that cluster and nodes.
  To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html)

* For APM monitoring:

To learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**), see
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).



