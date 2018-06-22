# SignalFx Agent - Default Docker Container Configuration

SignalFx bundleds a default agent.yaml within the [SignalFx docker image](https://docs.signalfx.com/en/latest/integrations/agent/overview.html#docker-image).
The following documentation describes how to interact with that configuration to emit metrics.

- (Configuration)[#configuration]
- (Running the Docker Image)[#running-the-docker-image]

## Configuration

### Required Configuration

The default configuration reads the following environment variables to dictate runtime configuration:

| Environment Variable  | Required | Description |
| --------- | -------- | ----------- |
| `SIGNALFX_ACCESS_TOKEN` | **yes** | The SignalFx API access token. |
| `SIGNALFX_INGEST_URL` | no (default:https://ingest.signalfx.com) | Often used in conjunction with the [SignalFx Metric Proxy](https://github.com/signalfx/metricproxy) to specify a different URL to where metrics are emitted |

### Additional Configuration Options

Both of the following configuration options require a modification or addition of files within
`/etc/signalfx`.  To make these files available to your docker container, mount a volume to `/etc/signalfx`
when starting your SignalFx docker container. 

See the [example run configuration](https://github.com/signalfx/signalfx-agent#docker-image)
 that mounts a volume for the docker container. 
 
#### Adding additional monitors

The default configuration will load any additional yaml files found in `/etc/signalfx/monitors/` as
part of the monitors configuration.  Specifically, the following snippet from the default `agent.yaml`
shows where the incorporated yaml files will be loaded:

```
monitors:
  - {"#from": "/etc/signalfx/monitors/*.yaml", flatten: true, optional: true}
  - type: collectd/cpu
```

For example, you can add an ElasticSearch monitor to a 
configuration by creating the following file within `/etc/signalfx/monitors/` that follows
 the (monitor config schema)[https://github.com/signalfx/signalfx-agent/blob/master/docs/monitor-config.md]:

```
- collectd/elasticsearch
  host: localhost
  port: 9200
```
Other options could be specified according to the (ElasticSearch Monitor configuration)[https://github.com/signalfx/signalfx-agent/blob/master/docs/monitors/collectd-elasticsearch.md]

#### Specifying a different agent.yaml

The configuration options above are a result of the default `agent.yaml` included in the 
SignalFx Docker image.  Including your own file named `agent.yaml` within the directory 
mounted to `/etc/signalfx/` will override this default file.  

## Running the Docker Image

See the example `docker run` command within the 
[installation instructions](https://github.com/signalfx/signalfx-agent#docker-image) for 
the Docker image. 

