# SignalFx NeoAgent

NeoAgent is an agent written in Go for monitoring nodes and application
services in highly ephemeral environments.

## Design Goals
Basic design goals are to have a minimal footprint with a plugin system so that
different monitoring agents like collectd can be embedded and dynamically
managed based on observed container activity in the underlying container
orchestration systems (kubernetes, mesos, and docker/swarm). These monitor and
observer plugins work with configuration templates and service matching rules
to support monitoring in an ephemeral environment. All SignalFx Collectd
Plugins are available to the agent (bundled) so any supported service that is
discovered can be automatically monitored. The agent will also include a set of
dimensions for each metric sent that associate each datapoint with the managing
orchestration system identifiers.

## Concepts

### Observers

Observers are what watch the various environments we support to discover running
services and automatically configure the agent to send metrics for those
services.

The observers we currently support are (follow links for more information):

 - **[File](./plugins/observers/file/file.go)**
 - **[Kubernetes](./plugins/observers/kubernetes/kubernetes.go)**
 - **[Mesosphere (Alpha)](./plugins/observers/mesosphere/mesosphere.go)**
 - **[Stand-alone Docker](./plugins/observers/docker/docker.go)**

### Monitors

Monitors are what collect metrics from services.  They can be configured either
manually or automatically by the observers.  Currently we rely on a
third-party "super monitor" called Collectd under the covers to do a lot of the
metric collection, although we also have monitors apart from Collectd.  They
are configured in the same way, however.

 - **[cAdvisor](./plugins/monitors/cadvisor/cadvisor.go)**
 - **[Kubernetes](./plugins/monitors/kubernetes/plugin.go)**


## Configuration

The agent is configured by a single configuration file: `agent.yaml`.

## Running

Right now, the agent is only provided as a Docker image. The agent's container
requires **privileged access** to the host node for both network and disk access.

### Logging
The default log level is `info`, which will log anything noteworthy in the
agent without spamming the logs too much.  Most of the `info` level logs are on
startup and upon service discovery changes.  `debug` will create very verbose
log output and should only be used when trying to resolve a problem with the
agent.

### Proxy Support

To use a HTTP(S) proxy, set the environment variable `HTTP_PROXY` and/or
`HTTPS_PROXY` in the container configuration to proxy either protocol.  The
agent will automatically manipulate the `NO_PROXY` envvar to not use the proxy
for local services.

### Kubernetes
* Configure secrets
    * Add a secret named `signalfx` that has a key `access-token` that is your SignalFX Access token.
    * Because the Quay repository is currently private you have to configure Docker registry authentication. Create a `docker-registry` type secret with name `quay-pull-secret` and in the data section set `.dockerconfigjson` to the base64 encoded contents of `~/.docker/config.json` (assuming you have already logged in with `docker login`)
* Create config maps:

        kubectl create -f deploy/kubernetes/signalfx-agent-configmap.yml \
                       -f deploy/kubernetes/signalfx-templates.yml
 then edit it as needed.
* Deploy the agent daemonset
    `kubectl create -f deploy/kubernetes/signalfx-agent.yml`

To override collectd templates modify the `signalfx-templates` config map.

##### Updating
Until we have an update script the easiest way to update the agents to a new version is to:

* `kubectl edit deploy signalfx-agent`
* Change the Docker image property (`.spec.template.spec.containers.image`) to the desired image
* Delete all agent pods (`kubectl delete pod -l app=signalfx-agent`) and they will be automatically recreated

### Mesos
```
TODO
```

## Diagnostics
The agent serves diagnostic information on a unix domain socket at
`/var/run/signalfx.sock`.  The socket takes no input, but simply dumps it's
current status back upon connection.  There is a small helper script
`agent-status` that is on the system PATH in the container that will call it
for you.

## Development

The agent can be built from a single multi-stage Dockerfile. This requires
Docker 17.06+.  Run `make image` to build the image using the build script
wrapper (`scripts/build.sh`).

There is a dev image that can be built for more convenient local development.
Run `make dev-image` to build it and `make run-dev-image` to run it.  It
basically just extends the standard agent image with some dev tools.  Within
this image, you can build the agent with `make signalfx-agent` and then run the
agent with `./signalfx-agent`.  You can put agent config in the `local-etc` dir
of this repo and it will be shared into the container (along with everything
else in this dir).

## Dependencies

Go dependencies are specified in `glide.yaml`. Of note the version of
docker/libkv is currently a forked version from
https://github.com/cohodata/libkv that has a ZooKeeper fix for watch events.

Run `make vendor` to pull down dependencies.

