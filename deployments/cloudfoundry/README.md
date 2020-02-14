# SignalFx CloudFoundry/Pivotal Platform Integrations

The SignalFx Smart Agent has three main uses within a Pivotal Platform environment:

1) **Loggregator Firehose Nozzle** - The agent has a monitor within it of type
`cf-firehose-nozzle` that connects to the [RLP
Gateway](https://github.com/cloudfoundry/loggregator/blob/master/docs/rlp_gateway.md)
service available in PCF 2.4+.  Our Ops Manager Tile will deploy the agent with
a prepackaged configuration that deploys an instance of the agent with this
monitor enabled and is horizontally scalable.

2) **Agent BOSH Release** - If you want to deploy the agent directly onto
deployment VMs, this is the way to go.  This can also be used as a [BOSH
addon](https://bosh.io/docs/runtime-config/#addons) if you want to deploy the
agent to every instance in the cluster.

3) **Application (PWS) Buildpack** - This is useful if you want to directly
monitor an application managed by Diego/PWS.  It runs the agent within the same
container as the application.  It can also be used to send APM trace spans from
an application to a local agent instance in order to obtain container correlation.
