# SignalFx Agent Docker Image

We provide a Docker image at
[quay.io/signalfx/signalfx-agent](https://quay.io/signalfx/signalfx-agent). The
image is tagged using the same agent version scheme.

If you are using Docker outside of Kubernetes, you can run the agent in a
Docker container and still gather metrics on the underlying host by running it
with the following flags:

```sh
$ docker run \
    --name signalfx-agent \
    --pid host \
    --net host \
    -v /:/hostfs:ro \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    -v /etc/signalfx/:/etc/signalfx/:ro \
    quay.io/signalfx/signalfx-agent:<version>
```

This assumes you have the agent config in the conventional directory
(`/etc/signalfx`) on the root mount namespace.  If you want to use a default,
built-in configuration, omit the volume bind mount for the `/etc/signalfx`
directory and see [Configuration](#configuration).

If you have the Docker API available through the conventional UNIX domain
socket, you should mount that in to be able to use the
[docker-container-stats](../../docs/monitors/docker-container-stats.md) monitor.

It is necessary to mount in the host root filesystem at `/hostfs` in order to
get disk usage metrics for the host filesystems using the [filesystems
monitor](../../docs/monitors/filesystems.md).  You will need to set the
`hostFSPath: /hostfs` config option on that monitor to make it use this
non-default path.

The other special config you will need is the `etcPath: /hostfs/etc` option
under the [host-metadata](../../docs/monitors/host-metadata.md) monitor config.
This tells it where to find certain files like `/etc/os-release` that are used
to generate host metadata such as the Linux distro and version.

By using the `--pid host` flag, the `/proc` filesystem in the container will
match the host's, so that no special configuration of the `/proc` path is
required.

You may also want to use the [Docker observer](../../docs/observers/docker.md) to
automatically discover other containers running in the same Docker engine.

## Configuration

For any non-trivial use-cases you will need to provide a custom configuration
file, but for simple setups or demo purposes, you can use [the supplied agent
configuration file](./agent.yaml) in the image by omitting the `-v
/etc/signalfx:/etc/signalfx/:ro` flag in the run command above.  Then you can
set the following environment variables on the agent container:

| Environment Variable  | Required | Description |
| --------- | -------- | ----------- |
| `SFX_ACCESS_TOKEN` | **yes** | The SignalFx API access token. |
| `SFX_INGEST_URL` | no | Often used in conjunction with the [SignalFx Gateway](https://github.com/signalfx/gateway) to specify a different target URL for datapoints and events. If not specified, this defaults to the global SignalFx ingest server. |
| `SFX_API_URL` | no | If you are operating in a different SignalFx realm, this value will need to be set to the SignalFx API server URL in your realm. |

The supplied configuration will also load any additional yaml files found in `/etc/signalfx/monitors/` as
part of the `monitors` list in the agent config like so:

```yaml
monitors:
  - {"#from": "/etc/signalfx/monitors/*.yaml", flatten: true, optional: true}
  - type: collectd/cpu
  ...
```

For example, you can add an ElasticSearch monitor to a configuration by
providing a Docker volume mount to `/etc/signalfx/monitors/` with a file that
follows the [monitor config schema](../../docs/monitor-config.md):

```yaml
- elasticsearch
  host: localhost
  port: 9200
```

Other options could be specified according to the [ElasticSearch Monitor
configuration](../../docs/monitors/elasticsearch.md)
