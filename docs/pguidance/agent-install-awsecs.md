# Install to AWS ECS

## Prerequisites

* Access to the Amazon Web Services (**AWS**) Elastic Container Services
(**ECS**) web console or the AWS Command Line Interface (CLI). To learn more,
refer to the AWS ECS documentation.
* A SignalFx access token. See [Smart Agent Access Token](https://docs.signalfx.com/en/latest/integrations/agent/access-token.html)

## Configure AWS ECS

First create the agent task definition for AWS ECS.

### AWS ECS web console

To create the agent task definition using the web admin console:

1. From the GitHub [ECS Deployment page](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ecs#ecs-deployment),
   download the file `signalfx-agent-task.json`.
2. Start the console and navigate to the **Task Definitions** tab.
3. Click **Create new Task Definition**.
4. Click **EC2**, then click **Next step**.

   **NOTE:** The Smart Agent only supports EC2 mode.
5. At the bottom of the page, click **Configure via JSON**.
6. Edit `signalfx-agent-task.json`, copy the contents, paste the contents into the text box, and click **Save**.
7. In the **Container Definitions** section, click the **signalfx-agent** container definition, then find the **environment variables** section.
8. Replace `ACCESS_TOKEN` with the value you obtained previously. See **Prerequisites**.
9. At the bottom of the task definition input form, click **Update** and then **Create**.

### AWS command-line interface

To create the agent task definition using the AWS command-line interface tool:

1. From the GitHub [ECS Deployment page](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ecs#ecs-deployment),
download the file `signalfx-agent-task.json`.
2. Run the following command:

        aws ecs register-task-definition --cli-input-json file:///<path_to_signalfx-agent-task.json>

## Configure the Smart Agent

By default, the agent container initialization script uses the agent configuration in
the file [agent.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/ecs/agent.yaml). The script
uses `curl` to pull this file from GitHub.

To provide other configuration options, set the environment variable `CONFIG_URL` in the agent task definition JSON file
to the URL of your custom configuration file. This location must be accessible from the ECS cluster.

The default configuration offers various env overrides that
you can set in the **environment variable** section of the agent task
definition file. Environment variable overrides have this form:

    {"#from": "env:VARNAME"...}  

To learn more, see [agent.yaml](https://github.com/signalfx/signalfx-agent/blob/master/deployments/ecs/agent.yaml).

## Installation

Run the Smart Agent as a Daemon service in an EC2 ECS cluster.

To create this service in the ECS web admin console:

1. In the console, go to your cluster.
2. Click the **Services** tab.
3. At the top of the tab, click **Create**.
4. Select the following options:
   - **Launch Type:** EC2
   - **Task Definition (Family):** signalfx-agent
   - **Task Definition (Revision):** <latest_revision>

     **NOTE:** If you haven't created a definition before, set this option to **1**.
   - **Service Name:** signalfx-agent
   - **Service type:** DAEMON
   - Use the defaults for the other options
5. Click **Next step**.
6. On this page, use the defaults for all options and click **Next step**.
7. On this page, use the defaults for all options and click **Next step**.
8. Click **Create Service**. AWS deploys the Smart Agent to each node in the ECS cluster.

## Verify the Smart Agent

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
  To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html).

* For APM monitoring:

To learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**), see
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).


