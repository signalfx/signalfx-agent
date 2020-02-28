# SignalFx Agent Puppet Module

This is a Puppet module that will install and configure the SignalFx Agent.  To
use it, simply include the class `signalfx_agent` in your manifests.  That
class accepts the following parameters:

 - `$config` (Hash): A config structure that gets directly converted to the agent
    YAML.  See the [Agent Config
    Schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md)
    for a full list of acceptable options.  The only required option is
    `signalFxAccessToken`.  Here is a basic config that will monitor a basic set of
    host-level components:

    ```ruby
    $config = {
      signalFxAccessToken: "MY_TOKEN",
      signalFxRealm: "us1",
      enableBuiltInFiltering: true,
      monitors: [
        {type: "collectd/cpu"},
        {type: "collectd/cpufreq"},
        {type: "collectd/df"},
        {type: "disk-io"},
        {type: "collectd/interface"},
        {type: "load"},
        {type: "collectd/memory"},
        {type: "collectd/protocols"},
        {type: "collectd/signalfx-metadata", "omitProcessInfo": true},
        {type: "host-metadata"},
        {type: "processlist"},
        {type: "collectd/uptime"},
        {type: "collectd/vmem"}
      ]
    }
    ```

	It is probably going to be simpler to keep this config in hiera at the path
	`signalfx_agent::config`, which will make it automatically filled in as a
	parameter.

    **Note:** In module version 0.4.0, the endpoint URLs have been removed from
    [default.yaml](./data/default.yaml). If upgrading the module from an older version,
    either the `signalFxRealm` or the endpoint URL options will need to be explicitly
    specified if using a realm other than `us0`.

 - `$package_stage`: The package repo stage to use: `release`, `beta`, or `test`
   (**default:** 'release')

 - `$config_file_path`: The path of the config file that will be rendered by the
   module (**default:** '/etc/signalfx/agent.yaml')

 - `$agent_version`: The agent release version, in the form `1.1.1`.  This
   corresponds to the [Github
   releases](https://github.com/signalfx/signalfx-agent/releases) _without_
   the `v` prefix. This option is **required** on Windows.

 - `$package_version`: The agent package version.  If not specified, for deb/rpm
   systems, this is automatically set to `<agent_version>-1` based on the
   `$agent_version` attribute above. For Windows, it is equivalent to the 
   agent version attribute. If set, `$package_version` will take precedence
   over `$agent_version`. On Windows, this option is not relevant.

 - `$installation_directory`: Valid only on Windows. The path where the SignalFx
   Agent should be downloaded to. (**default:** 'C:\\Program Files\\SignalFx\\')

## Dependencies

On Debian-based systems, the
[puppetlabs/apt](https://forge.puppet.com/puppetlabs/apt) module is required to
manage the SignalFx apt repository.

On Windows-based systems SignalFx Agent Puppet module has the following dependencies:

- [puppet/archive](https://forge.puppet.com/puppet/archive)

- [puppetlabs/powershell](https://forge.puppet.com/puppetlabs/powershell)

## Development

To work on the module in development, you can use the provided dev image to
test on Ubuntu 16.04, or for other machines, copy the module source to a
directory called `signalfx_agent` and then run:

```sh
puppet apply --modulepath <parent dir of signalfx_agent> -e 'class { signalfx_agent: 
  config => {
    signalFxAccessToken => 'test',
  }, agent_version => '1.1.1'
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
