# Install to Windows Using zip File

## Prerequisites

These prerequisites are for the host to which you're installing the Agent.

* Tasks

  Remove collector services such as `collectd`

  Remove third-party instrumentation and agent software
  **Note:**

  Do not use automatic instrumentation or instrumentation agents from
  other vendors when you're using SignalFx instrumentation. The results
  are unpredictable, but your instrumentation may break and your
  application may crash.

* Windows 8 or higher
* Windows PowerShell access
* Windows decompression application, such as **WinZip**
* .Net Framework 3.5 or higher
* Visual C++ compiler for Python
* Administrator account in which to run Smart Agent

## Steps

1. To get the latest Windows standalone installation package, navigate to
https://github.com/signalfx/signalfx-agent/releases, then download
`signalfx-agent-<latest_version>-win64.zip`.

For example, if the latest version is **5.1.6**:
- In the **releases** section, find the section entitled **v5.1.6**.
- In the **Assets** section, click `signalfx-agent-5.1.6-win64.zip`
- The `zip` file starts downloading.

2. Uncompress the `zip` file using your decompression application. The
package contents expand into the directory `signalfx-agent`.

## Configuration

Navigate to the `SignalFxAgent` directory, then create a configuration file for the agent:

- In a text editor, create a new file called `SignalFxAgent\agent-config.yml`
- In the file, add your host's hostname and port:
  `internalStatusHost: <local_hostname>`
  `internalStatusPort: <local_port>`
- Save the file

Smart Agent collects metrics based on the settings in
`agent-config.yml`. `internalStatusHost` and `internalStatusPort` specify
the host name and port of the host that's running the Smart Agent.

## Verification

### Start the Smart Agent

* To run the Smart Agent as a Windows program, run the following command in a console window:
`SignalFxAgent\bin\signalfx-agent.exe -config SignalFxAgent\agent-config.yml > <log_file>`

NOTE:The default log output for Smart Agent goes to `STDOUT` and `STDERR`. To persist log output, direct log output to <log_file>.

* To run Smart Agent as a Windows service:

- To install the agent as a service, run the following command in a console window:
`SignalFxAgent\bin\signalfx-agent.exe -service "install" -logEvents -config SignalFxAgent\agent-config.yml
- To start the agent service, run the following command in a console window:
`SignalFxAgent\bin\signalfx-agent.exe -service "start"`

To learn about other Windows service options, see [Service Configuration](https://docs.signalfx.com/en/latest/integrations/agent/windows.html#service-configuration).

### Verify the Smart Agent

To verify that your installation and config is working:

* For infrastructure monitoring:
  - In SignalFx UI, open the **Infrastructure** built-in dashboard
  - In the override bar at the top of the back, select **Choose a host**. Select one of your hosts from the dropdown.
  - The charts display metrics from your infrastructure.
 To learn more, see [Built-In Dashboards and Charts](https://docs.signalfx.com/en/latest/getting-started/built-in-content/built-in-dashboards.html).

* For Kubernetes monitoring:
  - In SignalFx UI, from the main menu select **Infrastructure** > **Kubernetes Navigator** > **Cluster map**.
  - In the cluster display, find the cluster you installed.
  - Click the magnification icon to view the nodes in the cluster.
  - The detail pane on the right hand side of the page displays details of your cluster and nodes.
  To learn more, see [Getting Around the Kubernetes Navigator](https://docs.signalfx.com/en/latest/integrations/kubernetes/get-around-k8s-navigator.html)

* For APM monitoring:

To learn how to install, configure, and verify the Smart Agent for Microservices APM (**µAPM**), see
[Overview of Microservices APM (µAPM)](https://docs.signalfx.com/en/latest/apm2/apm2-overview/apm2-overview.html).


