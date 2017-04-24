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

## Dependencies

Go dependencies are specified in `glide.yaml`. Of note the version of docker/libkv is currently a forked version from https://github.com/cohodata/libkv that has a ZooKeeper fix for watch events.

Run `glide install` to pull down dependencies.

## Build Image From Source

Run `make image`

## Run Agent Container

The agent's container requires privileged access to the host node for both network and disk access.
And on startup the agent reads an agent.yaml configuration to determine things like which plugins to load and basic file/data-access information.
This configuration can be set as an environment variable and should be based on the container orchestration system.

Here are examples of running agent:

### Kubernetes
* Configure secrets
    * Add a secret named `signalfx` that has a key `api-token` that is your SignalFX API token.
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
* Change the version numbers if applicable
* Change the Docker image property (`.spec.template.spec.containers.image`) to the desired image
* Delete all agent pods (`kubectl delete pod -l app=signalfx-agent`) and they'll be automatically recreated

### Mesos
```
TODO
```

### Local Docker - for development only

Here is an example of running signalfx-agent for local-docker using a docker compose file to start container and configure the agent.

Modify the example to work in your dev/test env
* Set the SFX_API_TOKEN envvar *required
* Add the SFX_HOSTNAME envvar to set the hostname (otherwise uses default behavior)
* Change the ingestUrls if you don't want to send to lab
* Set the SFX_MONITOR_USER to a test username.  or set to "".
* Set the SFX_MONITOR_PASSWORD to a test user password.  or set to "".

docker-compose.yml.
```
version: '2'
services:
  signalfx-agent:
    container_name: signalfx-agent
    image: quay.io/signalfuse/signalfx-agent:master
    privileged: true
    network_mode: host
    volumes:
     - /:/hostfs:ro
     - /etc/hostname:/mnt/hostname:ro
     - /etc:/mnt/etc:ro
     - /proc:/mnt/proc:ro
     - /var/run/docker.sock:/var/run/docker.sock
    environment:
     SFX_API_TOKEN: ${SFX_API_TOKEN}
     SFX_MONITOR_USER: ""
     SFX_MONITOR_PASSWORD: ""
     SET_FILE: /etc/signalfx/agent.yaml
     SET_FILE_CONTENT: |
        plugins:
            local-docker:
                plugin: observers/docker
                url: unix:///var/run/docker.sock

            collectd:
                plugin: monitors/collectd
                confFile: /etc/collectd/collectd.conf
                templatesDirs:
                - /etc/signalfx/collectd/templates
                pluginsDir: /usr/share/collectd
                staticPlugins:
                    writehttp-default:
                        plugin: writehttp
                        url: http://lab-ingest.corp.signalfuse.com:8080
                    signalfx-default:
                        plugin: signalfx
                        url: http://lab-ingest.corp.signalfuse.com:8080
                    docker-default:
                        plugin: docker
                        hostUrl: unix:///var/run/docker.sock

            debug:
                plugin: filters/debug

            service-mapping:
                plugin: filters/service-rules
                servicesFiles:
                - /etc/signalfx/collectd/custom-services.json
                - /etc/signalfx/collectd/services.json

        pipelines:
            docker:
            - local-docker
            - service-mapping
            - collectd
```

Note: Make sure to expose any container ports needed for monitoring to the host so they can be reached (required for local-docker only).
example: ```docker run -d -p 7099:7099 kafka```
