# SignalFx Agent Cookbook

This cookbook installs and configures the SignalFx Agent.

To install the agent, simply include the `signalfx_agent::default` recipe.  We
recommend pinning the agent to a specific version by setting the
`node['signalfx_agent']['agent_version']` attribute.  We will keep all old
versions in the repos.

The cookbook tries to be as flexible as possible with the configuration of the
agent and does not impose any agent configuration policy.  The default config
file (`/etc/signalfx/agent.yaml` on Linux) that comes from the package will be
overwritten with what you provide in the `node['signalfx_agent']['conf']`
object.

# Attributes

`node['signalfx_agent']['conf_file_path']`: The path where the agent config
will be rendered (default: `/etc/signalfx/agent.yaml` (Linux);
`\\ProgramData\SignalFxAgent\agent.yaml` (Windows))

`node['signalfx_agent']['agent_version']`: The agent release version, in the
form `1.1.1`.  This corresponds to the [Github
releases](https://github.com/signalfx/signalfx-agent/releases) _without_ the
`v` prefix.

`node['signalfx_agent']['package_version']`: The agent package version
(optional).  If not specified, for deb/rpm systems, this is automatically set
to `<agent_version>-1` based on the `node['signalfx_agent']['agent_version']`
attribute above.  For Windows, it is equivalent to the agent version attribute. 

`node['signalfx_agent']['package_stage']`: The package repository to use.  Can
be `release` (default, for main releases), `beta` (for beta releases), or `test`
(for unsigned test releases).

**Note:** SLES and openSUSE are only supported with cookbook versions 0.3.0 and newer,
and agent versions 4.7.7 and newer.

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
    {type: "disk-io"},
    {type: "collectd/interface"},
    {type: "load"},
    {type: "collectd/memory"},
    {"type": "collectd/signalfx-metadata", "omitProcessInfo": true},
    {type: "collectd/vmem"}
    {type: "host-metadata"},
    {type: "processlist"},
  ],
  "enableBuiltInFiltering": true
}
```

## Windows
This cookbook should work on Windows as well.  Note that we have come across
some issues with Python having a side-by-side manifest issue at times.  If this
is the case, make sure you have installed the [Microsoft Visual C++ Compiler
for Python 2.7](https://www.microsoft.com/EN-US/DOWNLOAD/DETAILS.ASPX?ID=44266) first.

## Development

To test this cookbook in the dev image (which is Ubuntu-based, so this won't be
able to test non-Debian packaging):

`chef-client -z -o 'recipe[signalfx_agent::default]' -j cookbooks/signalfx_agent/example_attrs.json`

When testing on a remote machine, put the contents of this directory into a
directory `cookbooks/signalfx_agent` located anywhere in the filesystem, create
a json attribute file with the desired attributes (see `example_attrs.json` for
an example), and then invoke chef-client as you would in the dev image.

## Release Process
To release a new version of the cookbook, run `./release` in this directory.
You will need to have our Chef Supermarket server `knife.rb` and the
`signalfx.pem` private key in `~/.chef`, which you can obtain from somebody
else on the project who has it.

You should update the version in `metadata.rb` to whatever is most appropriate
for semver and have that committed before running `./release`.

The release script will try to make and push an annotated tag of the form
`chef-vX.Y.Z` where `X.Y.Z` is the version in the `./metadata.rb` file.
