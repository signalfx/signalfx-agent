# Install Smart Agent using Configuration Management

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
* Org token (access token) from your SignalFx organization. To learn how
to obtain an org token, see [Working with access tokens](https://docs.signalfx.com/en/latest/admin-guide/tokens.html#working-with-access-tokens).
* Your SignalFx organization's realm. You can find this value in the SignalFx UI on your [profile page](https://docs.signalfx.com/en/latest/getting-started/get-around-ui.html#profile).

### Linux prerequisites

- Kernel version 2.6 or higher
- `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
- `terminal` or a similar command line interface application
- `puppetlabs/stdlib` module

### Debian prerequisites

- `CAP_DAC_READ_SEARCH` and `CAP_SYS_PTRACE` capabilities
- `terminal` or a similar command line interface application
- `puppetlabs/apt` module

### Windows prerequisites

- Windows 8 or higher
- Windows PowerShell access
- Microsoft Visual C++ Compiler for Python
-`puppet/archive` and `puppetlabs/powershell`

## Install using Puppet

The following sites host a Puppet module that installs the Smart Agent:

* [GitHub Agent Puppet Module](https://github.com/signalfx/signalfx-agent/tree/master/deployments/puppet)
* [Puppet Forge Agent Puppet Module](https://forge.puppet.com/signalfx/signalfx_agent)

### Configure for Puppet

Before you use Puppet to install the Agent, update your Puppet manifest with the class `signalfx_agent` and these
parameters:
* `$config`: The Smart Agent configuration. In this parameter, replace `<org_token>` and `<realm>` with the org token and realm values you
obtained previously. All other properties are optional. For example, this parameter represents a basic configuration that monitors host-level
components:

```
$config = {
  signalFxAccessToken: "<org_token>",
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
```
* `$package_stage`: The module version to use: 'release', 'beta', or 'test'. The default is 'release'.
* `$config_file_path`: The Smart Agent configuration file that the Puppet module uses to install the Agent. The default is `/etc/signalfx/agent.yaml`
* `$agent_version`: The agent release version, in the form n.n.n. Use the Smart Agent release version without the "v" prefix. This option is **required** on Windows.
For a list of the Smart Agent versions, see the [SignalFx Smart Agent releases page](https://github.com/signalfx/signalfx-agent/releases)
* `$package_version`: The agent package version. The default for Debian and RPM systems is 1 less than the value of `$agent_version`.
For Windows, the value is always `$agent_version`, so the Smart Agent ignores overrides. If set, `$package_version` takes precedence over `$agent_version`.
* `$installation_directory`: **Windows only**. The path to which Puppet downloads the Smart Agent.
The default is `C:\Program Files\SignalFx\`.
* $service_user and $service_group: **Linux only**. This parameter is only valid for agent package version 5.1.0 or higher.
It sets the user and group ownership for the `signalfx-agent` service. The user and group are created if they do not exist
The defaults are `$service_user = signalfx-agent` and `$service_group = signalfx-agent`.

To learn more about the Smart Agent configuration options,  
see the [Agent Configuration Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md).

### Installation

After you have your manifest updated, use Puppet to install the Smart Agent to your hosts.

### Verify your installation

See the following section [Verification](#verification)

## Install using Chef

The following sites host a Chef cookbook that installs the Smart Agent:

* [GitHub Smart Agent Chef Cookbook](https://github.com/signalfx/signalfx-agent/tree/master/deployments/chef#signalfx-agent-cookbook)
* [Chef Supermarket Smart Agent Cookbook](https://supermarket.chef.io/cookbooks/signalfx_agent)

### Configure for Chef

**NOTE:** SignalFx provides Chef support for SLES and openSUSE only with cookbook versions 0.3.0 and higher and agent versions 4.7.7 and higher.

Before you use Chef to install the Smart Agent, include the `signalfx_agent::default` recipe. Set the
`node['signalfx_agent']['agent_version']` attribute to the latest Smart Agent version listed on the
[SignalFx Smart Agent releases page](https://github.com/signalfx/signalfx-agent/releases).

Next, update the following attributes according to your system's configuration:

* `node['signalfx_agent']['conf']`: The Smart Agent configuration. This attribute becomes the agent configuration YAML file.
See the Agent Config Schema for a full list of acceptable options. In this parameter, replace `<org_token>` with the org token value you
obtained previously. All other properties are optional. For example, this attribute represents a basic configuration that monitors host-level components:

```
node['signalfx_agent']['conf'] = {
  signalFxAccessToken: "<org_token>",
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
```

* **Required** `node['signalfx_agent']['conf_file_path']`: Filename where Chef should put the agent configuration.
- For Linux, the default is `/etc/signalfx/agent.yaml`.
- For Windows, the default is `\ProgramData\SignalFxAgent\agent.yaml`.
* **Required** `node['signalfx_agent']['agent_version']`: The agent release version, in the form n.n.n. Use the
Smart Agent release version without the "v" prefix.
* Optional. `node['signalfx_agent']['package_version']`: The agent package version. The default for Debian and RPM systems is 1 less than the value of `$agent_version`.
For Windows, the default is `$agent_version`.
* **Required** `node['signalfx_agent']['package_stage']`: The recipe type to use: `release`, `beta`, or `test`. Test releases are unsigned.
* **Required** `node['signalfx_agent']['user'] and node['signalfx_agent']['group']`: **Linux only**. Only available for Agent version 5.1.0 or higher.
These attributes set the user and group ownership for the `signalfx-agent` service. The user or group (or both) are created if they don't exist.
The default value for both is `signalfx-agent`.

### Installation

After you add the Smart Agent recipe, use Chef to install the Smart Agent to your hosts.

### Verify your installation

See the following section [Verification](#verification)

## Install using Ansible

SignalFx provides an Ansible role for installing the Smart Agent:
* The main role site is the GitHub repo [SignalFx Agent Ansible Role](https://github.com/signalfx/signalfx-agent/tree/master/deployments/ansible).
To install the role from GitHub:
- Clone the repo to your controller.
- Add the `signalfx-agent` directory path to the `roles_path` in your `ansible.cfg` file.
* You can also get the role from Ansible Galaxy. To install the role from Galaxy, run the following command:
`ansible-galaxy install signalfx.smart_agent`

## Configure for Ansible

The Smart Agent Ansible role uses the following variables:

* `sfx_agent_config`: A mapping that Ansible converts to the Smart Agent configuration YAML file.
See the Agent Config Schema for a full list of acceptable options and their default values. The only required key-value pair is signalFxAccessToken.

    Here is a basic sfx_agent_config that will monitor a basic set of host-level components:

    sfx_agent_config:
      signalFxAccessToken: MY-TOKEN  # Required
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

    It is suggested to keep this mapping in a variable file in the respective group_vars or host_vars directory for your target remote hosts, or to pass it in via the -e @path/to/variable_file ansible-playbook extra vars option for a "global" configuration.

    sfx_config_file_path: The target path for the Smart Agent configuration file generated from sfx_agent_config (default: '/etc/signalfx/agent.yaml')

    sfx_repo_base_url: The url provided to yum/apt for obtaining the SignalFx Smart Agent (default: https://splunk.jfrog.io/splunk)

    sfx_package_stage: The package repo stage to use: release, beta, or test (default: 'release')

    sfx_version: The agent package version. This is of the form <agent version>-<package revision> (e.g. package version 3.0.1-1 is the first package revision that contains the agent version 3.0.1). Releases with package revision > 1 contain changes to some aspect of the packaging scripts (e.g. init scripts) but contain the same agent bundle. (default: 'latest')

    sfx_service_user and sfx_service_group: Set the user/group ownership for the signalfx-agent service. The user/group will be created if they do not exist. Requires agent package version 5.1.0 or newer. (default: 'signalfx-agent')

Note: After the signalfx-agent role is deployed, Ansible will manage the signalfx-agent service via the Ansible core service module. This module will automatically determine the host's init system for starting/stopping the signalfx-agent service, with a preference for systemd (systemctl).

## Install using Salt

## Install using a Docker image


## Verification