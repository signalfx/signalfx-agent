# SignalFx Agent BOSH Release and PCF Tile

This repo contains a BOSH Release of the [SignalFx Smart
Agent](https://github.com/signalfx/signalfx-agent). It also contains a Pivotal
Cloud Foundry Ops Manager tile definition which can be used to install a very
specific instance of the agent that will act as a Loggregator Firehose nozzle.

## Properties

See [./jobs/signalfx-agent/spec](./jobs/signalfx-agent/spec) for a full list of
properties available on the release.


## Development

During the development cycle when testing out changes, run the following:

```sh
# Make the latest agent tar bundle with `make bundle` or use one from a Github release.
# This only has to be run each time the tar bundle changes and not otherwise.
$ bosh add-blob ../../../signalfx-agent-latest.tar.gz signalfx_agent/signalfx-agent.tar.gz

$ bosh create-release --force --tarball ./latest-release.tgz --timestamp-version
```


## Releasing

See the [`release`](./release) script in this directory.

