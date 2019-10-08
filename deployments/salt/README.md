# SignalFx Agent formula

This formula installs and configures the SignalFx Agent.

To install the agent, simply include the signalfx-agent formula in `/srv/salt/` folder.
Rename and place the `pillar.example` in the `/srv/pillar/`.
Configure the `top.sls` in the `/srv/salt/` and `/srv/pillar/` accordingly.
We recommend pinning the agent to a specific version by setting the
`signalfx_agent.version` in the pillar.  We will keep all old
versions in the repos.

The formula tries to be as flexible as possible with the configuration of the
agent and does not impose any agent configuration policy.  The default config
file (`/etc/signalfx/agent.yaml`) that comes from the package will be
overwritten with what you provide in the pillar `signalfx-agent.conf`
object.

# Attributes

All the attributes can be configured in pillar
`signalfx-agent.conf_file_path`: The path where the agent config
 will be rendered (default: `/etc/signalfx/agent.yaml`)

`signalfx-agent.version`: The agent package version.  This is
of the form `<agent version>-<package revision>` (e.g. package version
`3.0.1-1` is the first package revision that contains the agent version
`3.0.1`).  Releases with package revision > 1 contain changes to some aspect of
the packaging scripts (e.g. init scripts) but contain the same agent bundle.

`signalfx-agent.package_stage`: The package repository to use.  Can
be `main` (default, for final releases), `beta` (for beta releases), or `test`
(for unsigned test releases).

`signalfx-agent.conf`: Agent configuration object.  Everything
underneath this object gets directly converted to YAML and becomes the agent
config file.  See the [Agent Config
Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md)
for a full list of acceptable options.  The only required option is
`signalFxAccessToken`.  Here is a basic config that will monitor a basic set of
host-level components:

```yaml
signalfx-agent:
  conf:
    signalFxAccessToken: 'My_Token'
    monitors:
      - type: collectd/cpu
      - type: collectd/cpufreq
      - type: filesystems
      - type: collectd/disk
      - type: collectd/interface
      - type: collectd/load
      - type: collectd/memory
      - type: collectd/signalfx-metadata
      - type: host-metadata
      - type: collectd/vmem
```

## Development

To test this formula in the dev image (which is Ubuntu-based, so this won't be
able to test non-Debian packaging):

Run Make file with following commands
`make dev-image` to create the docker image for development.
`make run-dev-image` to start the docker container with dev image and get into the container bash

`salt '*' state.apply` to run formula on the dev-image
