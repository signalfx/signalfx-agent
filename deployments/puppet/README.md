# SignalFx Agent Puppet Module

This is a Puppet module that will install and configure the SignalFx Agent.  To
use it, simply include the class `signalfx_agent` in your manifests.  That
class accepts the following parameters:

 - `$config` (Hash): A config structure that gets directly converted to the agent
    YAML.  See the [Agent Config
    Schema](../../docs/config-schema.md)
    for a full list of acceptable options.  The only required option is
    `signalFxAccessToken`.  Here is a basic config that will monitor a basic set of
    host-level components:

    ```ruby
    $config = {
      signalFxAccessToken: "MY_TOKEN",
      enableBuiltInFiltering: true,
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
        {type: "host-metadata"},
        {type: "collectd/uptime"},
        {type: "collectd/vmem"}
      ]
    }
    ```

	It is probably going to be simpler to keep this config in hiera at the path
	`signalfx_agent::config`, which will make it automatically filled in as a
	parameter.

 - `$package_stage`: The package repo stage to use: `final`, `beta`, or `test`
   (**default:** 'final')

 - `$config_file_path`: The path of the config file that will be rendered by the
   module (**default:** '/etc/signalfx/agent.yaml')

 - `$version`: The agent package version.  This is of the form `<agent
	 version>-<package revision>` (e.g. package version `3.0.1-1` is the first
	 package revision that contains the agent version `3.0.1`).  Releases with
	 package revision > 1 contain changes to some aspect of the packaging
	 scripts (e.g. init scripts) but contain the same agent bundle.

## Dependencies

On Debian-based systems, the
[puppetlabs/apt](https://forge.puppet.com/puppetlabs/apt) module is required to
manage the SignalFx apt repository.

## Development

To work on the module in development, you can use the provided dev image to
test on Ubuntu 16.04, or for other machines, copy the module source to a
directory called `signalfx_agent` and then run:

```sh
puppet apply --modulepath <parent dir of signalfx_agent> -e 'class { signalfx_agent: 
  config => {
    signalFxAccessToken => 'test',
  }
}'
```

If testing complex configurations, you can put the contents of the `-e` flag
into a file and pass that path as an argument to `puppet apply` instead.

## Release Process
To release a new version of the module, run `./release` in this directory.  You
will need access to the SignalFx account on the Puppet Forge website, and the
release script will give you instructions for what to do there.

You should update the version in `metadata.json` to whatever is most appropriate
for semver and have that committed before running `./release`.

The release script will try to make and push an annotated tag of the form
`puppet-vX.Y.Z` where `X.Y.Z` is the version in the `./metadata.json` file.
