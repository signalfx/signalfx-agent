# SignalFx Agent Ansible Role

This is a Ansible playbook contains `signalfx-agent` role that will install and configure the SignalFx Agent.  To
use it, simply include the role `signalfx-agent` in your play or run the playbook by passing the inventory.  That
playbook defines the following parameters:

 - `{{ conf }}` (Yaml): A config structure that gets directly converted to the agent
    YAML.  See the [Agent Config
    Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md)
    for a full list of acceptable options.  The only required option is
    `signalFxAccessToken`.  Here is a basic config that will monitor a basic set of
    host-level components:
    
    ```yaml
    conf:
      signalFxAccessToken: MY-TOKEN
      monitors:
        - type: collectd/cpu
        - type: collectd/cpufreq
        - type: collectd/df
        - type: collectd/disk
        - type: collectd/interface
        - type: collectd/load
        - type: collectd/memory
        - type: collectd/protocols
        - type: collectd/signalfx-metadata
        - type: collectd/uptime
        - type: collectd/vmem
    ```

	It is probably going to be simpler to keep this config in `target-group` var file at the 
	`group_vars` and update the same in playbook, which will make it visible to that group of hosts under inventory.

 - `{{ package_stage }}`: The package repo stage to use: `final`, `beta`, or `test`
   (**default:** 'final')

 - `{{ config_file_path }}`: The path of the config file that will be rendered by the
   module (**default:** '/etc/signalfx/agent.yaml')

 - `{{ version }}`: The agent package version.  This is of the form `<agent
	 version>-<package revision>` (e.g. package version `3.0.1-1` is the first
	 package revision that contains the agent version `3.0.1`).  Releases with
	 package revision > 1 contain changes to some aspect of the packaging
	 scripts (e.g. init scripts) but contain the same agent bundle.


## Development

To test this playbook in the dev image (which is Ubuntu-based, so this won't be
able to test non-Debian packaging):

`ansible-playbook -i <inventroy-file-path> playbook.yml`

When testing on a remote machine, put the contents of this directory into a
directory located anywhere in the filesystem, create
a inventory file with the desired target servers and update the signalfx-agent conf under `group_vars`, 
and then invoke `ansible-playbook` as you would in the dev image.
