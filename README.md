Neo is a SignalFx Agent written in Go for monitoring nodes and application services in highly ephemeral environments.

Basic design goals are to have a minimal footprint with a plugin system so
that different monitoring agents like collectd can be embedded and dynamically
managed based on observed container activity in the underlying container
orchestration systems (kubernetes, mesos, and docker/swarm). These monitor and
observer plugins work with configuration templates and service matching
rules to support monitoring in an ephemeral environment. All SignalFx Collectd
Plugins are available to the agent (bundled) so any supported service that is discovered
can be automatically monitored. The agent will also include a set of dimensions
for each metric sent that associate each datapoint with the managing orchestration
system identifiers.


## Build Image From Source

Run `build.sh`


## Run Agent Container

The agent's container requires privileged access to the host node for both network and disk access
And on startup the agent reads an agent.yaml configuration to determine things like which plugins to load and basic file/data-access information.
This configuration can be set as an environment variable and should be based on the container orchestration system.

Here are examples of running agent:

### Kubernetes

TODO

### Mesos

TODO

### Swarm

TODO

### Local Docker

The default configuration supports monitoring the local docker engine for services/containers and using embedded collectd as the monitoring agent.

Run signalfx-agent with the default configuration with the following command:
```
docker run --privileged \
  --net="host" \
  -e "SIGNALFX_API_TOKEN=XXXXXXXXXXXXXXXXXXXXXX" \
  -v /:/hostfs:ro \
  -v /etc/hostname:/mnt/hostname:ro \
  -v /etc:/mnt/etc:ro \
  -v /proc:/mnt/proc:ro \
  -v /var/run/docker.sock:/var/run/docker.sock \
  quay.io/signalfuse/signalfx-agent
```
