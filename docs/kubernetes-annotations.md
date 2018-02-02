# Config via K8s annotations

You can use Kubernetes pod annotations to tell the agent how to monitor your
services.  There are several annotations that the agent recognizes:

- `agent.signalfx.com/monitorType.<port>: "<monitor type>"` - Specifies the
	monitor type to use when monitoring the specified port.  If this value is
	present, any agent config will be ignored and so you must fully specify any
	non-default config values you want to use in annotations.  If this
	annotation is missing for a port but other config is present, you must have
	discovery rules or manually configured endpoints in your agent config to
	monitor this port (the other annotation config values will be merged into
	the agent config).

- `agent.signalfx.com/config.<port>.<configKey>: "<configValue>"` - Specifies
	a config option for the monitor that will monitor this endpoint.  The
	options are the same as specified in the monitor config reference.  Lists
	may be specified with the syntax `[a, b, c]` (YAML compact list) which
	will be deserialized to a list that will be provided to the monitor.
	Booleans values are simply the annotation string values `true` or
	`false`.  Integers can also be specified -- they must be strings as the
	annotation value, but they will be interpreted as an integer if they don't
	contain any non-number characters.

- `agent.signalfx.com/configFromEnv.<port>.<configKey>: "<env var name>"` --
	Specifies a config option that will be pulled from an environment variable
	on the same container as the port being monitored.

- `agent.signalfx.com/configFromSecret.<port>.<configKey>:
	"<secretName>/<secretKey>"` -- Maps the value of a secret to a config
	option.  The `<secretKey>` is the key of the secret value within the
	`data` object of the actual K8s Secret resource.  Note that this requires
	the agent's service account to have the correct permissions to read the
	specified secret.

In all of the above, the `<port>` field can be either the port number of the
endpoint you want to monitor or the assigned name.  The config is specific to a
single port, which allows you to monitor multiple ports in a single pod and
container by just specifying annotations with differing ports.

## Example

The following K8s pod spec and agent YAML configuration accomplish the same
thing:

K8s pod spec:

```yaml
    metadata:
      annotations:
        agent.signalfx.com/monitorType.jmx: "collectd/cassandra"
        agent.signalfx.com/config.jmx.intervalSeconds: "20"
        agent.signalfx.com/config.jmx.mBeansToCollect: "[cassandra-client-read-latency, threading]"
      labels:
        app: my-app
    spec:
      containers:
      - name: cassandra
        ports:
        - containerPort: 7199
          name: jmx
          protocol: TCP
       ......
```

Agent config:

```yaml
    monitors:
    - type: collectd/cassandra
      intervalSeconds: 20
      mBeansToCollect:
      - cassandra-client-read-latency
      - threading
```

If a pod has the `agent.signalfx.com/monitorType.*` annotation on it, that
pod will be excluded from the auto discovery mechanism and will be monitored
only with the given annotation configuration.  If you want to merge
configuration from the annotations with agent configuration, you must omit the
`monitorType` annotation and rely on auto discovery to find this endpoint.
At that point, config from both sources will be merged together, with pod
annotation config taking precedent.

