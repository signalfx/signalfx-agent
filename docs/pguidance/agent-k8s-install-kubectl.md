# Install using kubectl

Use kubectl to install the Smart Agent to Kubernetes environments.

If you have Helm installed already, you can also use Helm
to install the Smart Agent. To learn more, see
[Install the Smart Agent using Helm](agent-k8s-install-helm.md).

## Prerequisites

* Linux kernel version 2.6.32 or higher
* cap_dac_read_search and cap_sys_ptrace capabilities
* Terminal or a similar command-line interface application
* SignalFx access token. See [Smart Agent Access Token](../../../_sidebars-and-includes/smart-agent-access-token.html).
* Your SignalFx realm. See [Realms](../../../_sidebars-and-includes/smart-agent-realm-note.html).

## Configure kubectl for the Smart Agent

To install the Smart Agent using kubectl, you need to create a
Kubernetes secret for your access token and update settings in the Smart Agent's configuration files.

1. To create the Kubernetes secret, log in to the host from which you run kubectl.
2. Run the following command to create the Kubernetes secret called signalfx-agent, substituting `<access_token>` with
   your SignalFx access token:

   ```
   kubectl create secret generic --from-literal access-token=<access_token> signalfx-agent
   ```

3. On the same host, download the following configuration files from the
   [Kubernetes Deployments](https://github.com/signalfx/signalfx-agent/tree/master/deployments/k8s) area in GitHub:

   | File                      | Description                                                                                    |
   |:--------------------------|:-----------------------------------------------------------------------------------------------|
   | `clusterrole.yaml`        | Configuration settings for the ClusterRole cluster-admin. This file doesn't require an update. |
   | `clusterrolebinding.yaml` | Configuration settings for Kubernetes role-based access control (**RBAC**)                     |
   | `configmap.yaml`          | Cluster configuration settings                                                                 |
   | `daemonset.yaml`          | Daemonset configuration settings                                                               |
   | `serviceaccount.yaml`     | Configuration settings for Kubernetes Service Accounts. This file doesn't require an update.   |

4. Update `configmap.yaml`:
   - Cluster name: For each of your Kubernetes clusters, replace `MY-CLUSTER` with a unique cluster name.
   - Realm: Update the value of `signalFxRealm` with the name of your SignalFx realm.
   - To avoid sending docker and cadvisor metrics from some containers,
     update the `datapointsToExclude` property. To learn more, see [Filtering](https://docs.signalfx.com/en/latest/integrations/agent/filtering.html#filtering).
5. In the clusterrolebinding.yaml file, update `MY_AGENT_NAMESPACE` or the service account token reference with the Smart
   Agent namespace in which you're deploying the agent.

6. Update the daemonset.yaml file:

   - For RBAC-enabled clusters, add the permissions required for the Smart Agent.

   - For Rancher nodes, ensure that the Docker engine proxy is configured so that it can pull the `signalfx-agent` Docker image from `quay.io`.
     To learn more, see the Rancher v1.6 or Rancher v2.x documentation regarding proxy configuration.

   - Update the **cAdvisor** monitor configuration to use port 9344:

     ```
     monitors:
       - type: cadvisor
       - cadvisorURL: http://localhost:9344
     ```

7. If you're using OpenShift and you can't use the default namespace, modify each
   namespace occurrence and then ask your cluster administrator to run the following commands:

   ```
   oc create serviceaccount signalfx-agent
   oc adm policy add-cluster-role-to-user anyuid system:serviceaccount:default:signalfx-agent
   oc edit scc privileged
   ```

   Make the following changes to the scc file:

   ```
   users:
    - system:serviceaccount:default:signalfx-agent
    - serviceAccountName: signalfx-agent
   ```

## Install the Smart Agent

1. Remove collector services such as `collectd`.

2. Remove third-party instrumentation and agent software.

> Do not use automatic instrumentation or instrumentation agents from
> other vendors when you're using SignalFx instrumentation. The results
> are unpredictable, but your instrumentation may break and your
> application may crash.

3. Run the following command to update `kubectl` with the configuration files you've just modified:

   ```
   cat *.yaml | kubectl apply -f-
   ```

## Verify the Smart Agent

After you install the Smart Agent, it starts sending data from your clusters to SignalFx.

To see the services the Smart Agent has discovered, run the following command inside any of your Smart Agent containers:

```
while read -r line; do kubectl exec --namespace `echo $line` signalfx-agent status; done <<< `kubectl get pods -l app=signalfx-agent --all-namespaces --no-headers | tr -s " " | cut -d " " -f 1,2`
```

In addition, you can do the following in SignalFx:

* For infrastructure monitoring:
  1. In SignalFx, open the **Infrastructure** built-in dashboard.
  2. In the override bar, select **Choose a host**. Select one of your nodes from the dropdown.

  The charts display metrics from the infrastructure for that node.

  To learn more, see [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

* For Kubernetes monitoring:
  1. In SignalFx, from the main menu select **Infrastructure** > **Kubernetes Navigator** > **Cluster map**.
  2. The map displays all the clusters running the Smart Agent.
  3. Click the magnification icon to view the nodes in a cluster.

  The detail pane displays details of that cluster and nodes.

  To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html).

* For APM monitoring, learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**). See
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).
