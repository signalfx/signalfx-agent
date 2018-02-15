# SignalFx Agent

The SignalFx Agent is a metric agent written in Go for monitoring nodes and
application services in a variety of different environments.

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

* Add a secret named `signalfx` that has a key `access-token` that is your SignalFX Access token.

* Create the config maps:

	kubectl create -f ./deployments/k8s/signalfx-agent.configmap.yml

 then edit it as needed.

* Deploy the agent daemonset:

    kubectl create -f ./deployments/k8s/signalfx-agent.daemonset.yml

To override collectd templates modify the `signalfx-templates` config map.


## Diagnostics
The agent serves diagnostic information on a unix domain socket at
`/var/run/signalfx.sock`.  The socket takes no input, but simply dumps it's
current status back upon connection.  The command `signalfx-agent status` (or
the special symlink `agent-status`) will read this socket and dump out its
contents.

## Development

The agent is built from a single multi-stage Dockerfile. This requires Docker
17.06+.  There is a dev image that can be built for more convenient local
development. Run `make dev-image` to build it and `make run-dev-image` to run
it and attach to a shell inside of it.  Inside this dev image, the agent bundle
is at `/bundle` and the rest of the image contains useful tools for
development, such as a golang build environment.

Within this image, you can build the agent with `make signalfx-agent` and then
run the agent with `./signalfx-agent`.  The code directory will be mounted in
the container at the right place in the go path so that it can be built with no
extra setup.

You can put agent config in the `local-etc` dir of this repo and it will be
shared into the container at the default place that the agent looks for config
(`/etc/signalfx`).

### Trivial Commits
If you have a very simple commit that should not require a full CI run, just
put the text `[skip ci]` in the commit message somewhere and CircleCI will not
run for that commit.

### Development in Kubernetes (K8s)

* Because the Quay repository is currently private you have to configure Docker registry authentication. Create a `docker-registry` type secret with name `quay-pull-secret` and in the data section set `.dockerconfigjson` to the base64 encoded contents of `~/.docker/config.json` (assuming you have already logged in with `docker login`)

* `kubectl edit deploy signalfx-agent`
* Change the Docker image property (`.spec.template.spec.containers.image`) to the desired image
* Delete all agent pods (`kubectl delete pod -l app=signalfx-agent`) and they will be automatically recreated


## Dependencies

We are using [dep](https://github.com/golang/dep) to manage dependencies.  It
isn't quite GA yet but seems to meet our needs sufficiently.

Run `make vendor` to pull down dependencies.

