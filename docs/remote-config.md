# Remote Configuration

The agent always requires a main config file on a local filesystem, but values
within that config can pull from other sources.  These sources include other
files on the filesystem or KV stores such as Zookeeper, Etcd, and Consul.
Additional stores can be easily added.

A remote config value looks like this in the main config file:

```yaml
signalFxAccessToken: {"#from": "/etc/signalfx/token"}
```

The remote config specification is a YAML map object that includes the `#from`
key.  The value of that key is the path from which to get the config value.

The value of the `#from` key has the form `<source>:<path>`.  If it is only a
path with no `<source>:`, it is assumed to be a file path on the local
filesystem.  All non-filesystem sources must be configured in the
`sourceConfig` section of the main config file.

If the source is a reference to a single path, that value is deserialized by
YAML and inserted into the config as if it had been literally in the config as
such.  The replacement is done in such a way that you don't need to worry
about matching indentation of remote values.

## Simple paths

The most basic way to reference a value is to use a single, non-globbed path.
The content stored at that path can be simple scalar values such as strings or
integers, or YAML collections such as sequences or maps.  The only requirement
is that they deserialize from YAML properly.  Note that JSON is a subset of
YAML, so any valid JSON can also be used.

## Globbed paths

If there is a glob in the source path, the YAML content of the matching paths
will be read and deserialized.  All of the values must be YAML collections of
the same type (i.e. either all a sequence or all a map), or else an error is
raised.  Then all of those collections will be merged together and treated as
one collection.

### Flattening

You can flatten both sequences and maps into the parent of the item where the
remote value is specified.  For example:

```yaml
monitors:
 - type: collectd/mysql
   username: signalfx
   databases:
    - name: admin
    - {"#from": "zk:/signalfx-agent/mysql/databases/*", flatten: true}
```

Given the following znodes name and values in ZooKeeper:

 - `/signalfx-agent/mysql/databases/app1`: `[{name: my-db1}, {name: my-db2}]`
 - `/signalfx-agent/mysql/databases/app2`: `[{name: their-db1}, {name: their-db2}]`

The final resolved config would look like this:

```yaml
monitors:
 - type: collectd/mysql
   username: signalfx
   databases:
    - name: admin
    - name: my-db1
    - name: my-db2
    - name: their-db1
    - name: their-db2
```

### Optional paths
You may want to allow for globbed paths that don't actually match anything.
This is very useful for specifying a directory of extra monitor configurations
that may be empty such as the following:

```yaml
signalFxAccessToken: abcd
monitors:
 - {"#from": "/etc/signalfx/conf2/*.yaml", flatten: true, optional: true}
 - type: collectd/cpu
 - type: collectd/cpufreq
 - type: collectd/df
```

The key here is the `optional: true` value which makes it accept globs that
don't match anything.  `optional` defaults to `false` so it must be explicitly
stated that you are ok with no matches.

`optional` also works in scalar contexts as well, assuming that the config value
is not required by the agent.

### Raw Values
If you have values in files/KV stores that you don't want interpreted as YAML,
but rather as plain strings, you can add the `raw: true` option to the remote
value specification.  Everything else acts as it would otherwise.

## Environment Variables

The config file also supports environment variable interpolation with the
`${VARNAME}` syntax (the curly brackets are required).  Environment variables
are interpolated *after* all of the remote config values are interpolated,
which means that remote config values can contain references to envvars, if so
desired.  Envvars cannot, however, contain remote config values.

## Other

If you need more sophisticated interpolation of config values from KV stores,
we recommend using a third-party templating solution such as
[confd](https://github.com/kelseyhightower/confd/), or rolling your own
scripting.

