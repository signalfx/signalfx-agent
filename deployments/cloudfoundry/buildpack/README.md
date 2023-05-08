# SignalFx Smart Agent CloudFoundry Buildpack

A [CloudFoundry buildpack](https://docs.vmware.com/en/VMware-Tanzu-Application-Service/2.11/tas-for-vms/toc-buildpacks-index.html) to install
and run the SignalFx Smart Agent in PWS (Diego managed) apps.  This will
probably work for generic CloudFoundry apps as well, but it is only tested and
supported on Pivotal Platform.

## Installation

**This buildpack is created automatically when the SignalFx Montioring tile is
installed via Ops Manager on Pivotal Platform.**  That is the preferred
installation route.

If you are using plain CloudFoundry and would like to install the buildpack,
clone this repo and change to this directory, then run:

```sh
# Add buildpack for SignalFx Agent
$ cf create-buildpack signalfx_agent_buildpack . 99 --enable
```

```sh
# Basic setup, see Configuration for more envvars that can be set
$ cf set-env my-app SIGNALFX_ACCESS_TOKEN <my org token>
$ cf set-env my-app SIGNALFX_REALM <my org realm>


$ cf v3-push my-app -b signalfx_agent_buildpack -b <main_buildpack>
```

## Configuration

**The following only applies if you are using the "built-in" agent.yaml config
provided by the buildpack.  If you provide a custom agent.yaml in your
application (and refer to it in the sidecar configuration), these might not
work unless you have preserved the references to the envvars in the `#from`
remote config blocks.**

Set the following environment variables with `cf set-env` as applicable to configure this buildpack:

 - `SIGNALFX_REALM` - The realm to which to send metrics (e.g. `us1`)
 - `SIGNALFX_INGEST_URL` - The base URL of the SignalFx ingest server to use (automatically derived from `SIGNALFX_REALM` if set)
 - `SIGNALFX_API_URL` - The base URL of the SignalFx API server to use (automatically derived from `SIGNALFX_REALM` if set)
 - `SIGNALFX_ACCESS_TOKEN` - Your SignalFx org access token
 - `SIGNALFX_AGENT_VERSION` - Version of the SignalFx Agent to be configured
   (e.g. `4.19.2`). The buildpack depends on features present in version
   4.19.2+.


## Sidecar Configuration

The recommended method for running the agent is to run it as a sidecar using
the CloudFoundry [sidecar
functionality](https://docs.cloudfoundry.org/devguide/sidecars.html).
Additional information can be found [in the v3 API
docs](http://v3-apidocs.cloudfoundry.org/version/release-candidate/#sidecars).

Here is an example application `manifest.yml` file that would run the agent as
a sidecar:

```yaml
applications:
  - name: my-app
    disk_quota: 1G
    instances: 1
    memory: 1G
    sidecars:
      - name: signalfx-agent
        process_types: ['web']
        command: '.signalfx/signalfx-agent/bin/signalfx-agent -config .signalfx/etc/agent.yaml'
        memory: 75MB
```

