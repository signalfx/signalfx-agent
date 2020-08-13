# Fargate Deployment

## Create Task Definition
The agent is designed to be run as a sidecar in a task with Fargate containers
to be monitored.  This means you must add a Smart Agent container into each
task definition for applications that you wish to monitor with the agent.

If you want to use a common configuration of agent and have the service
auto-discovered, make sure all your Fargate containers to monitor have Docker
labels to specify ports to be monitored in your task definition:

```json
"containerDefinitions": [
    {
        "name": "my-container",
        ...
        "dockerLabels": {
           "agent.signalfx.com.port.6379": "true",
           "agent.signalfx.com.monitorType.6379": "collectd/redis",
           "agent.signalfx.com.config.6379.intervalSeconds": "1"
        }
    },
    ...
]
```

The label `agent.signalfx.com.port.<port>: "true"` specifies port number to be
autodiscovered on the Fargate container.  There is no other way for the agent
to know about these port via the autodiscovery process.

The label `agent.signalfx.com.monitorType.<port>: "<monitor type>"` specifies
the [monitor
type](https://docs.signalfx.com/en/latest/integrations/agent/monitor-config.html#monitor-list)
to use to monitor this endpoint.  You can specify other config for that monitor
using labels similar to the `"agent.signalfx.com.config.6379.intervalSeconds":
"1"` one shown in the example above.  The format is
`agent.signalfx.com.config.<port>.<config key>: "<value as string">`.  The
value string is interpreted as YAML, so you can use compact YAML notation to
specify non-scalar values.

We have an [example task definition](./example-fargate-task.json) that shows
launching the agent alongside a Redis cache instance.  The example uses
auto-discovery, but you could also just hard code the monitor configuration in
the agent.yaml (or reference config from envvars using remote config, as shown
in [agent.yaml](./agent.yaml)).


## Configuration

The main technique for configuring the agent is to have a config file
downloaded from the network using curl in the agent container's initialization
script.  By default it pulls from [the config file in our Github
repository](./agent.yaml) that provides a basic config that might suffice for
basic monitoring cases.  If you wish to provide a more complex config file you
can set the `CONFIG_URL` env var in the agent task definition to the URL of the
config file.  This location must be accessible from the ECS cluster.

The default config supports various environment variable overrides, which you
can set in the environment variable section of the agent task definition.  See
[agent.yaml](./agent.yaml) for details (hint: it is the config values that are
of the form `{"#from": "env:VARNAME"...}`).

The agent will not send any metrics about itself by default
[agent.yaml](./agent.yaml) configuration although it is also a running
container. If you wish to see the metrics about the agent, you can remove
`signalfx-agent` from `excludedImages` config in [agent.yaml](./agent.yaml).
