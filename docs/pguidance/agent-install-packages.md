# Install Using Unix Packages

Install SignalFx Smart Agent using either a Debian or RPM package.

## Install using the Debian Package

### Prerequisites for Debian

* Unix distro that's based on Debian or supports Debian packages
* Kernel version 2.6 or higher
* `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
* APT or similar package tools. These instructions show you how to install the package using `apt-get`.
* Internet access. If necessary, set up proxies to allow your package tools to access the Internet.
* `terminal` or a similar command line interface application
* Permission to run `curl` and `sudo`

### Install with Debian

1. Remove collector services such as `collectd`

2. Remove third-party instrumentation and agent software
   **Note:**
   Do not use automatic instrumentation or instrumentation agents from
   other vendors when you're using SignalFx instrumentation. The results
   are unpredictable, but your instrumentation may break and your
   application may crash.

3. To download the GNU Privacy Guard (**GnuPG**) security key for the Debian package, run

        curl -sSL https://splunk.jfrog.io/splunk/signalfx-agent-deb/splunk-B3CD4420.gpg > /etc/apt/trusted.gpg.d/splunk.gpg

4. To add an entry for the SignalFx Smart Agent package to Debian, run

        echo 'deb https://splunk.jfrog.io/splunk/signalfx-agent-deb release main' > /etc/apt/sources.list.d/signalfx-agent.list

5. To update the Debian package lists with the SignalFx Smart Agent package information, run

        apt-get update

6. To install the Agent, run

        apt-get install -y signalfx-agent

### Configure Debian installation

Set Smart Agent configuration options in the configuration YAML file. To learn more,
see [Agent Configuration](../config-schema.md).

### Verify Debian installation

See the section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Install using RPM

### Prerequisites for RPM

These prerequisites are for the host to which you're installing the Agent:

* Unix distribution that supports RPM, such as a Linux distribution or CentOS
* Kernel version 2.6 or higher
* `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
* `yum` or similar package tools. These instructions show you how to install the package using `yum`.
* Write access to `/etc/yum.repos.d/`
* Internet access. If necessary, set up proxies to allow your package tools to access the Internet.
* `terminal` or a similar command line interface application

### Install with RPM

1. Remove collector services such as `collectd`

2. Remove third-party instrumentation and agent software
   **Note:**
   Do not use automatic instrumentation or instrumentation agents from
   other vendors when you're using SignalFx instrumentation. The results
   are unpredictable, but your instrumentation may break and your
   application may crash.

3. In the directory `/etc/yum.repos.d/`, edit a new file named `signalfx-agent.repo`.
4. Add these lines to the file:

        [signalfx-agent]
        name=SignalFx Agent Repository
        baseurl=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/release
        gpgcheck=1
        gpgkey=https://splunk.jfrog.io/splunk/signalfx-agent-rpm/splunk-B3CD4420.pub
        enabled=1

5. Run the following command:

        yum install -y signalfx-agent

### Configure RPM installation

Set Smart Agent configuration options in the configuration YAML file. To learn more,
see [Agent Configuration](../config-schema.md).

### Verify RPM install

See the section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Verify the Smart Agent

For Debian or RPM, to verify that your installation and configuration are working:

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
