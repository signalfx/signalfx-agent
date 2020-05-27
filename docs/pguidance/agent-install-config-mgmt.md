# Install Using Configuration Management

Install the Smart Agent using one of the following configuration management  
packages:



| Package | Section                                                          |
|---------|------------------------------------------------------------------|
| Puppet  | [Install using a Puppet module](#install-using-a-puppet-module)  |
| Chef    | [Install using a Chef cookbook](#install-using-a-chef-cookbook)  |
| Ansible  | [Install using an Ansible role](#install-using-an-ansible-role) |
| Salt    | [Install using a Salt formula](#install-using-a-salt-formula)    |
| Docker  | [Install using a Docker image](#install-using-a-docker-image)    |

## Prerequisites

### Prerequisites for all platforms

* Your SignalFx realm. See [Realms](https://docs.signalfx.com/en/latest/integrations/agent/realm-note.html).
* A SignalFx access token. See [Smart Agent Access Token](https://docs.signalfx.com/en/latest/integrations/agent/access-token.html)

### *nix prerequisites

* Kernel version 2.6 or higher
* `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
* `terminal` or a similar command line interface application

### Windows prerequisites

- Windows 8 or higher
- Windows PowerShell access
- Microsoft Visual C++ Compiler for Python

## Install using a Puppet module

Before you install, configure the Smart Agent Puppet manifest

### Configure for Puppet

1. Download the Puppet module from one of these sites:

   * [GitHub Agent Puppet Module](https://github.com/signalfx/signalfx-agent/tree/master/deployments/puppet)
   * [Puppet Forge Agent Puppet Module](https://forge.puppet.com/signalfx/signalfx_agent)

2. From the downloaded module, add the Smart Agent manifest to your Puppet installation.
3. Update your Puppet manifest with the class `signalfx_agent` and these parameters:
   
   - `$config`: The Smart Agent configuration. In this parameter, replace `<access_token>` and `<realm>` with the access token and realm
     values you obtained previously, as described in the section [Prerequisites for all platforms](#prerequisites-for-all-platforms).
     All other properties are optional. For example, the following parameter represents a basic configuration that monitors
     host-level components:

             $config = {
              signalFxAccessToken: "<access_token>",
              signalFxRealm: "<realm>",
              monitors: [
                {type: "cpu"},
                {type: "filesystems"},
                {type: "disk-io"},
                {type: "net-io"},
                {type: "load"},
                {type: "memory"},
                {type: "host-metadata"},
                {type: "processlist"},
                {type: "vmem"}
              ]
            }

   - `$package_stage`: The module version to use: 'release', 'beta', or 'test'. The default is 'release'.

   - `$config_file_path`: The Smart Agent configuration file that the Puppet module uses to install the Agent.
     The default is `/etc/signalfx/agent.yaml`
   - `$agent_version`: The agent release version, in the form n.n.n. Use the Smart Agent release version without the "v" prefix.
     This option is **required** on Windows. For a list of the Smart Agent versions, see the
     [SignalFx Smart Agent releases page](https://github.com/signalfx/signalfx-agent/releases)
   - `$package_version`: The agent package version. The default for Debian and RPM systems is 1 less than the value of `$agent_version`.
     For Windows, the value is always `$agent_version`, so the Smart Agent ignores overrides. If set,
     `$package_version` takes precedence over `$agent_version`.
   - `$installation_directory`: **Windows only**. The path to which Puppet downloads the Smart Agent.
     The default is `C:\Program Files\SignalFx\`.
   - `$service_user` and `$service_group`: **Linux only**. This parameter is only valid for agent package version 5.1.0 or higher.
     It sets the user and group ownership for the `signalfx-agent` service. The user and group are created if they do not exist
     The defaults are `$service_user = signalfx-agent` and `$service_group = signalfx-agent`.

To learn more about the Smart Agent configuration options,
see the [Agent Configuration Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md).

### Install with Puppet

1. Remove collector services such as `collectd`

   Remove third-party instrumentation and agent software
   **Note:**

   Do not use automatic instrumentation or instrumentation agents from
   other vendors when you're using SignalFx instrumentation. The results
   are unpredictable, but your instrumentation may break and your
   application may crash.
2. Confirm that you have the correct Puppet tool for your platform:

   - For Linux: `puppetlabs/stdlib` module
   - For Debian systems: `puppetlabs/apt` module
   - For Windows: `puppet/archive` and `puppetlabs/powershell`

3. Use Puppet to install the Smart Agent to your hosts.

### Verify your Puppet installation

See the following section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Install using a Chef cookbook

Before you install, configure the Smart Agent Chef cookbook.

### Configure for Chef

1. Download the Smart Agent Chef cookbook from one of these sites:

   - [GitHub Smart Agent Chef Cookbook](https://github.com/signalfx/signalfx-agent/tree/master/deployments/chef#signalfx-agent-cookbook)
   - [Chef Supermarket Smart Agent Cookbook](https://supermarket.chef.io/cookbooks/signalfx_agent)

   **NOTE:**  
   SignalFx provides Chef support for SLES and openSUSE  
   starting with cookbook versions 0.3.0 and higher and agent versions 4.7.7 and  
   higher.

2. Include the `signalfx_agent::default` recipe. Set the `node['signalfx_agent']['agent_version']`
   attribute to the latest Smart Agent version listed on the
   [SignalFx Smart Agent releases page](https://github.com/signalfx/signalfx-agent/releases).

3. Update these attributes to match your system's configuration:

   * `node['signalfx_agent']['conf']`: The Smart Agent configuration. This attribute
     becomes the agent configuration YAML file.

     See the Agent Config Schema for a full list of acceptable options. In this attribute,
     replace `<access_token>` with the access token value you obtained previously, as described in  
     the section [Prerequisites for all platforms](#prerequisites-for-all-platforms).  
     All other properties for this attribute are optional. For example, this attribute
     represents a basic configuration that monitors host-level components:

            node['signalfx_agent']['conf'] = {
              signalFxAccessToken: "<access_token>",
              monitors: [
                {type: "cpu"},
                {type: "filesystems"},
                {type: "disk-io"},
                {type: "net-io"},
                {type: "load"},
                {type: "memory"},
                {type: "vmem"}
                {type: "host-metadata"},
                {type: "processlist"},
              ]
            }

   * **Required** `node['signalfx_agent']['conf_file_path']`: File name where Chef should put the agent configuration.
     - For Linux, the default is `/etc/signalfx/agent.yaml`.
     - For Windows, the default is `\ProgramData\SignalFxAgent\agent.yaml`.
   * **Required** `node['signalfx_agent']['agent_version']`: The agent release version, in the form n.n.n. Use the
     Smart Agent release version without the "v" prefix.
   * **Required** `node['signalfx_agent']['package_stage']`: The recipe type to use: `release`, `beta`, or `test`.
     Test releases are unsigned.
   * **Required** `node['signalfx_agent']['user'] and node['signalfx_agent']['group']`:
   * **Only required for Linux**. These attributes are only available for Agent version 5.1.0 or higher.
     They set the user and group ownership for the `signalfx-agent` service. The user or group (or both) are
     created if they don't exist. The default value for both is `signalfx-agent`.
   * Optional. `node['signalfx_agent']['package_version']`: The agent package version.
     - For systems that use Debian or RPM, the default is 1 less than the value of `$agent_version`.
     - For Windows, the default is `$agent_version`.

To learn more about the Smart Agent configuration options,
see the [Agent Configuration Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md).

### Install with Chef

After you add the Smart Agent recipe, use Chef to install the Smart Agent to your hosts.

### Verify your Chef installation

See the following section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Install using an Ansible role

SignalFx provides an Ansible role for installing the Smart Agent:

The main role site is the GitHub repo [SignalFx Agent Ansible Role](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).

To install the role from GitHub:

1. Clone the repo to your controller.
2. Add the `signalfx-agent` directory path to the `roles_path` in your `ansible.cfg` file.

   You can also get the role from Ansible Galaxy. To install the role from Galaxy, run the following command:

        ansible-galaxy install signalfx.smart_agent

### Configure for Ansible

The Smart Agent Ansible role uses the following variables:

* `sfx_agent_config`: A mapping that Ansible converts to the Smart Agent configuration YAML file.
  See the Agent Config Schema for a full list of acceptable options and their default values.
  In this mapping, replace `<access_token>` with the access token value you obtained previously, as described in the
  section [Prerequisites for all platforms](#prerequisites-for-all-platforms).
  All other properties are optional.

  For example, this mapping monitors basic host-level components:

        sfx_agent_config:
            signalFxAccessToken: <access_token>  # Required
            monitors:
            - type: cpu
            - type: filesystems
            - type: disk-io
            - type: net-io
            - type: load
            - type: memory
            - type: vmem
            - type: host-metadata
            - type: processlist

   Keep your mapping in a custom file in your target remote host's `group_vars` or `host_vars` directory,
   or pass it to Ansible using the `-e @<path_to_file>` ansible-playbook extra vars option for a global configuration.

* `sfx_config_file_path`: Destination path for the Smart Agent configuration file generated from the `sfx_agent_config` mapping.
  The default is `/etc/signalfx/agent.yaml`.
* `sfx_repo_base_url`: URL for the SignalFx Smart Agent repo. The default is `https://splunk.jfrog.io/splunk)`.
* `sfx_package_stage`: Module version to use: `release`, `beta`, or `test`. The default is `release`.
* `sfx_version`: Desired agent version, specified as `<agent version>-<package revision>`. For example,
  `3.0.1-1` is the first package revision that contains the agent version 3.0.1.
  Releases with package revision > 1 contain changes to some aspect of the packaging scripts, such as the `init` scripts, but
  contain the same agent bundle, which defaults to 'latest'.
* `sfx_service_user`, `sfx_service_group`: Set the user and group for the `signalfx-agent` service.
  They're created if they don't exist. This property is only available in agent package version 5.1.0 or higher.
  The default value for both properties is `signalfx-agent`.

Note: After deploying the signalfx-agent role, Ansible manages the `signalfx-agent` service  
using the Ansible core service module. This module automatically determines the  
host's init system for starting and stopping the signalfx-agent service, with a preference for `systemd` (`systemctl`).

To learn more about the Smart Agent configuration options,
see the [Agent Configuration Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md).

### Install with Ansible

After you install the Ansible role, use Ansible to install the Smart Agent to your hosts.

### Verify your Ansible installation

See the following section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Install using a Salt formula

SignalFx provides a Salt formula that installs and configures the Smart Agent on Linux.
Download the formula from the [SignalFx Agent Salt formula site](https://github.com/signalfx/signalfx-agent/tree/master/deployments/salt).

### Configure for Salt

1. Download the Smart Agent formula to `/srv/salt`.
2. Copy `pillar.example` from the download to `/srv/pillar` and rename it to `pillar`.
3. Update `top.sls` in `/srv/salt` and `/srv/pillar` to point to the Smart Agent formula.
4. Update the new `pillar` with these attributes:
   - **Required:** `signalfx-agent.conf`: The agent configuration object. Replace `<access_token>` with the access token value you
     obtained previously, as described in [Prerequisites for all platforms](#prerequisites-for-all-platforms).
     For example, this configuration object monitors basic host-level components:

            signalfx-agent:
              conf:
                signalFxAccessToken: '<access_token>'
                monitors:
                  - type: cpu
                  - type: filesystems
                  - type: disk-io
                  - type: net-io
                  - type: load
                  - type: memory
                  - type: vmem
                  - type: host-metadata
                  - type: processlist

   - Optional: `signalfx-agent.version`: Desired agent version, specified as `<agent version>-<package revision>`. For example,
     `3.0.1-1` is the first package revision that contains the agent version 3.0.1. Releases with package revision > 1
     contain changes to some aspect of the packaging scripts, such as the `init` scripts, but
     contain the same agent bundle, which defaults to 'latest'.
   - Optional: `signalfx-agent.package_stage`: Module version to use: `release`, `beta`, or `test`. The default is `release`.
     Test releases are unsigned.
   - Optional: `signalfx-agent.conf_file_path`: Destination file for the Smart Agent configuration file generated by the installation.
     The installation overwrites the `agent.yaml` downloaded in the Salt formula with the values specified by the `signalfx-agent.conf`
     attribute in `pillar`. The default destination is `/etc/signalfx/agent.yaml`.
   - Optional: `signalfx-agent.service_user`, `signalfx-agent.service_group`: Set the user and group for the `signalfx-agent` service.
     They're created if they don't exist. This property is only available in agent package version 5.1.0 or higher.
     The default value for both properties is `signalfx-agent`.

To learn more about the Smart Agent configuration options,  
see the [Agent Configuration Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md).

### Install with Salt

After you configure the Smart Agent Salt formula, Salt installs the Smart Agent on your hosts.

### Verify your Salt installation

See the following section entitled [Verify the Smart Agent](#verify-the-smart-agent).

## Install using a Docker image

SignalFx hosts a Smart Agent Docker image at  
[https://quay.io/signalfx/signalfx-agent](https://quay.io/signalfx/signalfx-agent).
This image is tagged with the same values as the Smart Agent itself. For example, to get version 5.1.6 of the Smart Agent,
download the Docker image with the tag 5.1.6.

Install using the Docker image if you're using Docker **without** Kubernetes. To install the Smart Agent to Docker containers
running Kubernetes:

* If you use Helm: [Install using Helm](agent-k8s-install-helm.md)
* If you use kubectl: [Install using kubectl](agent-k8s-install-kubectl.md).

### Configure for Docker

To configure the Docker image for the Smart Agent:

1. In the container for the Smart Agent, set the following environment variables:
   - `SFX_ACCESS_TOKEN`: Access token value you obtained previously. See [Prerequisites for all platforms](#prerequisites-for-all-platforms).
   - `SFX_API_URL`: SignalFx API server URL. This value has the following syntax:
     `https://api.<realm>.signalfx.com`. Replace `<realm>` with the realm value you obtained previously,
     as described in [Prerequisites for all platforms](#prerequisites-for-all-platforms).
   - `SFX_INGEST_URL`: SignalFx ingest URL. Set this if you're using the Smart Gateway or a different target for
     datapoints and events.
2. In the agent configuration file `agent.yaml` that's downloaded with the  Docker image,
   update any incorrect property values to match your system.
3. In `agent.yaml`, add additional properties such as monitors and observers. To learn
   more about all the available configuration options, see the [Agent Configuration Schema](../config-schema.md).
4. In your agent container, copy `agent.yaml` to the directory `/etc/signalfx/`.
5. If you have the Docker API available through the conventional UNIX domain socket, mount it so
   you can use the [docker-container-stats](../monitors/docker-container-stats.md) monitor.
6. To determine the agent version you want to run, see [SignalFx Smart Agent Releases](https://github.com/signalfx/signalfx-agent/releases).
   Unless SignalFx advises you to do otherwise, choose the latest version.

To configure optional monitors, add the following lines to `agent.yaml`:

1. To load monitor configuration YAML files from `/etc/signalfx/monitors/`, add the following line to
   the `monitors` property in `agent.yaml`:

        monitors:
          [omitted lines]
          - {"#from": "/etc/signalfx/monitors/*.yaml", flatten: true, optional: true}

2. To get disk usage metrics for the host filesystems using the [filesystems](../monitors/filesystems.md) monitor:
   - Mount the `hostfs` root file system.
   - Add the `filesystems` monitor to `agent.yaml`:

        procPath: /hostfs/proc
        [omitted lines]
        monitors:
          [omitted lines]
          - type: filesystems
            hostFSPath: /hostfs```

3. Add the **host metadata** monitor to `agent.yaml`:

        etcPath: /hostfs/etc
        [omitted lines]
        monitors:
          - type: host-metadata

To learn more about configuring monitors and observers for the Smart Agent in Docker, see
[Agent Configuration](../config-schema.md).

### Install in Docker

To install and start the Smart Agent in a Docker container, run the following command, replacing `<version>` with  
Smart Agent version number you obtained previously, as described in [Prerequisites for all platforms](#prerequisites-for-all-platforms):

        docker run \
          --name signalfx-agent \
          --pid host \
          --net host \
          -v /:/hostfs:ro \
          -v /var/run/docker.sock:/var/run/docker.sock:ro \
          -v /etc/signalfx/:/etc/signalfx/:ro \
          -v /etc/passwd:/etc/passwd:ro \
          quay.io/signalfx/signalfx-agent:<version>

### Verify your Docker installation

See the following section entitled [Verify the Smart Agent](#verify-the-smart-agent).

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
