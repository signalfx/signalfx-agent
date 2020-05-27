# Install Using Helm

## Prerequisites

* Linux kernel version 2.6 or higher
* `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
* `terminal` or a similar command line interface application
* Helm:

  - The installation chart for the Smart Agent is compatible with Helm version 2 or Helm version 3.
  - If you haven't already installed Helm, install Helm version 3.
  - To learn more, see the [Helm site](https://helm.sh/).

* Tiller installed on all of your Kubernetes hosts. Because of
  the way that the installation chart works, you need Tiller even if you're using
  Helm version 3.
* Your SignalFx realm. See [Realms](../../../_sidebars-and-includes/realm-note.html).
* A SignalFx access token. See [Smart Agent Access Token](../../../_sidebars-and-includes/access-token.html)


## Configure Helm for the Smart Agent

The SignalFx Smart Agent chart for Helm comes with a `values.yaml` configuration
file that contains useful default values. If you want to modify these values,
override them by creating your own "values" file. Follow these steps:

1. Download the [default values file](https://github.com/signalfx/signalfx-agent/blob/master/deployments/k8s/helm/signalfx-agent/values.yaml).
2. Rename the file. For example, rename it to `myValues.yaml`.
3. Edit the file to update the values with your own choices, then save the file.

When you install the Smart Agent using Helm, add the parameter `-f <values_yaml_file>` to the install command.
For example, add the parameter `-f myValues.yaml`.

## Install with Helm

1. Remove collector services such as `collectd`
2. Remove third-party instrumentation and agent software

   **Note:**

   Do not use automatic instrumentation or instrumentation agents from
   other vendors when you're using SignalFx instrumentation. The results
   are unpredictable, but your instrumentation may break and your
   application may crash.

3. To add the SignalFx Helm chart repository to Helm, enter the following command:

        helm repo add signalfx https://dl.signalfx.com/helm-repo

4. To ensure that the repository is up-to-date, enter the following command:

        helm repo update

5. Determine the following values:

| Name                 | Example                  | Meaning                                                         |
|----------------------|--------------------------|-----------------------------------------------------------------|
| `<values_yaml_file>` | `myValues.yaml`          | Optional. YAML file containing your configuration values¹       |
| `<access_token>`     | '-zz9a_Z9z99ZZzZZZZ-ZZz' | **REQUIRED**. Access token. See [Prerequisites](#prerequisites) |
| `<cluster_name>`     | 'myCluster'              | **REQUIRED**.Name of the cluster to monitor                     |
| `<version>`          | 5.1.6                    | Optional. Version of the Smart Agent you want to use²           |
| `<realm>`            | 'us0'                    | **REQUIRED**. Your realm. See [Prerequisites](#prerequisites)   |

¹ If you don't specify a values file, Helm installs the defaults. If you don't use this parameter,
  don't use `-f`.
² See [Smart Agent releases](https://github.com/signalfx/signalfx-agent/releases) for
  a list of releases. If you don't specify a release number, Helm installs the latest release. If you don't
  use this parameter, don't use `--set agentVersion=`.

4. To install the Smart Agent

   If you want to have OpenShift support, substitute the values from the previous step and run this command :

       helm install -f <values_yaml_file> --set signalFxAccessToken=<access_token> --set clusterName=<cluster_name> --set agentVersion=<version> --set signalFxRealm=<realm> signalfx/signalfx-agent --set kubernetesDistro=openshift

   If you *don't* want to have OpenShift support, substitute the values from the previous step and run this command:

       helm install -f <values_yaml_file> --set signalFxAccessToken=<access_token> --set clusterName=<cluster_name> --set agentVersion=<version> --set signalFxRealm=<realm> signalfx/signalfx-agent

### Verify the Smart Agent

The Smart Agent runs as soon as you install it using Helm.

To verify that your installation and config is working:

* For infrastructure monitoring:
  - In SignalFx UI, open the **Infrastructure** built-in dashboard
  - In the override bar at the top of the back, select **Choose a host**. Select one of your nodes from the dropdown.
  - The charts display metrics from the infrastructure for that node.
    To learn more, see [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

* For Kubernetes monitoring:
  - In SignalFx UI, from the main menu select **Infrastructure** > **Kubernetes Navigator** > **Cluster map**.
  - In the cluster display, find the cluster `<cluster_name>` you chose in the previous steps.
  - Click the magnification icon to view the nodes in the cluster.
  - The detail pane on the right hand side of the page displays details of your cluster and nodes.
    To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html)

* For APM monitoring:

To learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**), see
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).


