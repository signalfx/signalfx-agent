# SignalFx Agent Cookbook

This cookbook installs and configures the SignalFx Agent.

To install the agent, simply include the `signalfx_agent::default` recipe.  We
recommend pinning the agent to a specific version by setting the
`node['signalfx_agent']['package_version']` attribute.  We will keep all old
versions in the repos.

The cookbook tries to be as flexible as possible with the configuration of the
agent and does not impose any agent configuration policy.  The default config
file (`/etc/signalfx/agent.yaml`) that comes from the package will be
overwritten with what you provide in the `node['signalfx_agent']['conf']`
object.

# Attributes

`node['signalfx_agent']['conf_file_path']`: The path where the agent config
 will be rendered (default: `/etc/signalfx/agent.yaml`)

`node['signalfx_agent']['package_version']`: The agent package version.  This is
of the form `<agent version>-<package revision>` (e.g. package version
`3.0.1-1` is the first package revision that contains the agent version
`3.0.1`).  Releases with package revision > 1 contain changes to some aspect of
the packaging scripts (e.g. init scripts) but contain the same agent bundle.

`node['signalfx_agent']['package_stage']`: The package repository to use.  Can
be `final` (default, for main releases), `beta` (for beta releases), or `test`
(for unsigned test releases).

`node['signalfx_agent']['conf']`: Agent configuration object.  Everything
underneath this object gets directly converted to YAML and becomes the agent
config file.  See the [Agent Config
Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md)
for a full list of acceptable options.  The only required option is
`signalFxAccessToken`.  Here is a basic config that will monitor a basic set of
host-level components:

```ruby
node['signalfx_agent']['conf'] = {
  signalFxAccessToken: "MY_TOKEN",
  monitors: [
    {type: "collectd/cpu"},
    {type: "collectd/cpufreq"},
    {type: "collectd/df"},
    {type: "collectd/disk"},
    {type: "collectd/interface"},
    {type: "collectd/load"},
    {type: "collectd/memory"},
    {type: "collectd/protocols"},
    {type: "collectd/signalfx-metadata"},
    {type: "collectd/uptime"},
    {type: "collectd/vmem"}
  ]
}
```

## Development

To test this cookbook in the dev image (which is Ubuntu-based, so this won't be
able to test non-Debian packaging):

`chef-client -z -o 'recipe[signalfx_agent::default]' -j cookbooks/signalfx_agent/example_attrs.json`

When testing on a remote machine, put the contents of this directory into a
directory `cookbooks/signalfx_agent` located anywhere in the filesystem, create
a json attribute file with the desired attributes (see `example_attrs.json` for
an example), and then invoke chef-client as you would in the dev image.
