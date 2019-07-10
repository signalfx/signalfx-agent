# Monitor Consul Connect with Envoy Proxy

Consul Connect emits metrics in statsd format to help understand the network. To report these metrics to SignalFx, you need to configure the Consul Connect proxy to send the metrics to a StatsD sink exposed by the agent.

First, configure Consul Connect Proxy to emit StatsD metrics.
See the example Consul service definition below:

```hcl
services {
  name = "my_service"
  port = 8080
  connect {
    sidecar_service {
      proxy {
        config {
          envoy_extra_stats_sinks_json = <<EOF
{
  "name": "envoy.statsd", 
  "config": {
    "address": {
      "socket_address": {
        "address": "127.0.0.1",
        "port_value": 8125,
        "protocol": "UDP"
      }
    },
    "prefix": "consul.my_service_mesh.my_service"
  }
}
EOF
        }
      }
    }
  }
}
services {
  name = "my_service2"
  port = 9080
  connect {
    sidecar_service {
      proxy {
        config {
          envoy_extra_stats_sinks_json = <<EOF2
{
  "name": "envoy.statsd", 
  "config": {
    "address": {
      "socket_address": {
        "address": "127.0.0.1",
        "port_value": 8125,
        "protocol": "UDP"
      }
    },
    "prefix": "consul.my_service_mesh.my_service2"
  }
}
EOF2
        }
      }
    }
  }
}
```

It is mandatory to provide mesh/service name through a prefix as shown since Consul does not specify the source as part of the metric name. The source information will be parsed into dimensions by setting patterns in the agent configuration. For example:

```yaml
monitors:
  - type: statsd
    listenPort: 8125
    # metrics need to be tagged with `plugin: consul`
    extraDimensions:
      plugin: consul
    converters:
    - pattern: "consul.{mesh}.{service}.cluster.{}.{action}"
      metricName: "{action}"
```

For more information about StatsD monitor configuration, see the documentation on [StatsD monitor](https://github.com/signalfx/signalfx-agent/blob/master/docs/monitors/statsd.md).
