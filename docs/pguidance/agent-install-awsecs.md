# Install to AWS EC2

Deploy the SignalFx Smart Agent to an AWS EC2 instance using a SignalFx
configuration script, and run the Smart Agent as a Daemon service in an
EC2 cluster.

## Prerequisites

* Access to the Amazon Web Services (AWS) Elastic Compute Cloud
  (EC2) web console or the AWS Command Line Interface (CLI). To learn more,
  refer to the AWS EC2 documentation.
* SignalFx access token. See [Smart Agent Access Token](https://docs.signalfx.com/en/latest/integrations/agent/smart-agent-access-token.html).
* Your SignalFx realm. See [Realms](../../../_sidebars-and-includes/smart-agent-realm-note.html).

## Configure the Smart Agent for AWS EC2

Configure the Smart Agent for AWS EC2 by following these steps:

1. [Edit the main configuration file](#edit-the-main-configuration-file)
2. Optional: [Edit additional options](#edit-additional-options)
Create a Smart Agent task definition for AWS EC2.

### Edit the main configuration file

1. Download the [signalfx-agent-task.json](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ecs/signalfx-agent-task.json) file.
2. Edit the file and make these replacements:

| Text                    | Replacement                                                                                                                                                                                                                                                     |
|:------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MY_ACCESS_TOKEN`       | Your SignalFx access token. See [Prerequisites](#prerequisites).                                                                                                                                                                                                |
| `MY_INGEST_URL`         | `https://ingest.<REALM>.signalfx.com`. Replace `<REALM>` with your realm. See [Prerequisites](#prerequisites)                                                                                                                                                   |
| `MY_API_URL`            | `https://api.<REALM>.signalfx.com`. Replace `<REALM>` with your realm. See [Prerequisites](#prerequisites)                                                                                                                                                      |
| `MY_TRACE_ENDPOINT_URL` | `null`. The Smart Agent only uses the property for Microservices APM. To learn more about the Smart Agent and Microservice APM, see [Deploy a SignalFx Smart Agent for µAPM](https://docs.signalfx.com/en/latest/apm/apm-getting-started/apm-smart-agent.html). |

### Edit additional options

By default, the main configuration in signalfx-task-agent.json uses additional options in the
agent.yaml file by pulling them from GitHub using `curl`. These options control how the Smart Agent
interacts with EC2. For example, the `observer` option specifies which features the Smart Agent
uses to discover running services.

To change additional configuration options, follow these steps:

1. Download the [agent.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/ecs/agent.yaml) file.
2. Copy agent.yaml to a new .yaml file with a custom name.
3. In the signalfx-agent-task.json file, change the environment variable `CONFIG_URL` to the URL of your
   custom version of agent.yaml. The URL must be accessible from your EC2 cluster.
4. Deploy the custom .yaml file to your EC2 cluster.

To learn more, see [agent.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/ecs/agent.yaml).

## Deploy the Smart Agent task definition to EC2

After you finish editing the configuration files, continue with these steps:

* If you already use the AWS EC2 web console, use it to create the task definition
* If you're not using the web console, use the command-line interface to create the task definition

### AWS EC2 web console

1. Start the web console and navigate to the **Task Definitions** tab.
2. Click **Create new Task Definition**.
3. Click **EC2**, then click **Next step**.
4. Click **Configure via JSON**.
5. Open the signalfx-agent-task.json file, copy the contents, paste the contents into the text box, and click **Save**.
6. Click **Update** and then **Create**.

### AWS command-line interface

Create the agent task definition using the AWS command-line interface tool by entering the following command:

```
aws ecs register-task-definition --cli-input-json file:///<path_to_signalfx-agent-task.json>
```

## Installation

Run the Smart Agent as a Daemon service in an EC2 cluster.

To create this service in the EC2 web admin console:

1. In the console, go to your cluster.
2. Click the **Services** tab.
3. Click **Create**.
4. Select the following options:
   - Launch Type: EC2
   - Task Definition (Family): signalfx-agent
   - Task Definition (Revision): <latest_revision>

     If you haven't created a definition before, set this option to **1**.

   - Service Name: signalfx-agent
   - Service type: DAEMON
   - Use the defaults for the other options
5. Click **Next step**.
6. Use the defaults for all options and click **Next step**.
7. Use the defaults for all options and click **Next step**.
8. Click **Create Service**. AWS deploys the Smart Agent to each node in  the EC2 cluster.

## Verify the Smart Agent

* For infrastructure monitoring, perform these steps:
  1. In SignalFx, open the **Infrastructure** built-in dashboard
  2. In the override bar at the top of the back, select **Choose a
     host**. Select one of your nodes from the dropdown list.

  The charts display metrics from the infrastructure for that node.

  To learn more, see [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

* For Kubernetes monitoring, perform these steps:
  1. In SignalFx, from the main menu select **Infrastructure** > **Kubernetes Navigator** > **Cluster map**.
  2. The map displays all the clusters running the Smart Agent
  3. Click the magnification icon to view the nodes in a cluster.

  The detail pane on the right hand side of the page displays details of that cluster and nodes.

  To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html).

* For APM monitoring, learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**). See
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).


