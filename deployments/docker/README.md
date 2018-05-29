# SignalFx Agent - Default Docker Container Configuration

SignalFx bundleds a default agent.yaml within the [SignalFx docker image](https://docs.signalfx.com/en/latest/integrations/agent/overview.html#docker-image).
The following documentation describes how to interact with that configuration to emit metrics.

## Configuration

The configuration options below expect that you've mounted a volume at `/etc/signalfx/`.  The default 
`agent.yaml` reads files from this volume to affect its runtime configuration. 

See the example `docker run` command within the 
[installation instructions](https://github.com/signalfx/signalfx-agent#docker-image) for 
the Docker image. 


| File  | Required | Description |
| --------- | -------- | ----------- |
| `/etc/signalfx/token` | **yes** | The SignalFx API access token is read from this file. |
| `/etc/signalfx/ingest_url` | no (default:https://ingest.signalfx.com) | Often used in conjunction with the [SignalFx Metric Proxy](https://github.com/signalfx/metricproxy) to specify a different URL to where metrics are emitted |
| `/etc/signalfx/monitors/*.yaml` | no | Monitors specified in file(s) at this path will be loaded into the configuration. See [monitor config schema](https://github.com/signalfx/signalfx-agent/blob/master/docs/config-schema.md#monitors) for expected syntax. |

### Specifying a different agent.yaml

The configuration options above are a result of the default `agent.yaml` included in the 
SignalFx Docker image.  Including your own file named `agent.yaml` and mounted to `/etc/signalfx/` 
will override this default file.  