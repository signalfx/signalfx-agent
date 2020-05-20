
<!--- Generated by to-integrations-repo script in Smart Agent repo, DO NOT MODIFY HERE --->
<!--- GENERATED BY gomplate from scripts/docs/templates/monitor-page.md.tmpl --->

# collectd/haproxy

Monitor Type: `collectd/haproxy` (<a target="_blank" href="https://github.com/signalfx/signalfx-agent/tree/master/pkg/monitors/collectd/haproxy">Source</a>)

**Accepts Endpoints**: **Yes**

**Multiple Instances Allowed**: Yes

## Overview

This monitors an <a target="_blank" href="http://www.haproxy.org/">HAProxy</a> instance.  Requires HAProxy 1.5+.

<!--- SETUP --->
### Socket Config
The location of the HAProxy socket file is defined in the HAProxy config file, as in the following example:

```
global
    daemon
    stats socket /var/run/haproxy.sock
    stats timeout 2m
```

Note: it is possible to use a tcp socket for stats in HAProxy. Users will
first need to define in their collectd-haproxy plugin config file the tcp
address for the socket, for example `localhost:9000`, and then in the
haproxy.cfg file change the stats socket to listen on the same address
```
global
    daemon
    stats socket localhost:9000
    stats timeout 2m
```

For a more restricted tcp socket, a backend server can be defined to listen
to stats on localhost. A frontend proxy can use the backend server on a
different port, with ACLs to restrict access. See below for example.
Depending on how restrictive your socket is, you may need to add the
signalfx-agent user to the haproxy group:
`sudo usermod -a -G haproxy signalfx-agent`

```
global
    daemon
    stats socket localhost:9000
    stats timeout 2m

backend stats-backend
    mode tcp
    server stats-localhost localhost:9000

frontend stats-frontend
    bind *:9001
    default_backend stats-backend
    acl ...
    acl ...
```

<!--- SETUP --->
### SELinux Setup

If you have SELinux enabled, create a SELinux policy package downloading
the <a target="_blank" href="https://github.com/signalfx/collectd-haproxy/blob/master/selinux/collectd-haproxy.te">type enforcement
file</a>
to some place on your server.  Run the commands below to create and install
the policy package.

    $ checkmodule -M -m -o collectd-haproxy.mod collectd-haproxy.te
    checkmodule:  loading policy configuration from collectd-haproxy.te
    checkmodule:  policy configuration loaded
    checkmodule:  writing binary representation (version 17) to collectd-haproxy.mod
    $ semodule_package -o collectd-haproxy.pp -m collectd-haproxy.mod
    $ sudo semodule -i collectd-haproxy.pp
    $ sudo reboot


## Configuration

To activate this monitor in the Smart Agent, add the following to your
agent config:

```
monitors:  # All monitor config goes under this key
 - type: collectd/haproxy
   ...  # Additional config
```

**For a list of monitor options that are common to all monitors, see [Common
Configuration](./_monitor-config.html#common-configuration).**


| Config option | Required | Type | Description |
| --- | --- | --- | --- |
| `pythonBinary` | no | `string` | Path to a python binary that should be used to execute the Python code. If not set, a built-in runtime will be used.  Can include arguments to the binary as well. |
| `host` | **yes** | `string` |  |
| `port` | no | `integer` |  (**default:** `0`) |
| `proxiesToMonitor` | no | `list of strings` | A list of all the pxname(s) or svname(s) that you want to monitor (e.g. `["http-in", "server1", "backend"]`) |
| `excludedMetrics` | no | `list of strings` | Deprecated.  Please use `datapointsToExclude` on the monitor config block instead. |
| `enhancedMetrics` | no | `bool` |  (**default:** `false`) |


## Metrics

These are the metrics available for this monitor.
Metrics that are categorized as
[container/host](https://docs.signalfx.com/en/latest/admin-guide/usage.html#about-custom-bundled-and-high-resolution-metrics)
(*default*) are ***in bold and italics*** in the list below.


 - `counter.connection_total` (*counter*)<br>    Cumulative number of connections (frontend). This corresponds to HAProxy's "conn_tot" metric.
 - ***`counter.server_selected_total`*** (*counter*)<br>    Number of times a server was selected, either for new sessions or when re-dispatching. This corresponds to HAProxy's "lbtot" metric.
 - ***`derive.bytes_in`*** (*cumulative*)<br>    Corresponds to HAProxy's `bin` metric -  Bytes in
 - ***`derive.bytes_out`*** (*cumulative*)<br>    Corresponds to HAProxy's `bout` metric -  Bytes out
 - `derive.cli_abrt` (*cumulative*)<br>    Corresponds to HAProxy's `cli_abrt` metric -  Number of data transfers aborted by the client
 - `derive.comp_byp` (*cumulative*)<br>    Corresponds to HAProxy's `comp_byp` metric -  Number of bytes that bypassed the HTTP compressor (CPU/BW limit)
 - `derive.comp_in` (*cumulative*)<br>    Corresponds to HAProxy's `comp_in` metric -  Number of HTTP response bytes fed to the compressor
 - `derive.comp_out` (*cumulative*)<br>    Corresponds to HAProxy's `comp_out` metric -  Number of HTTP response bytes emitted by the compressor
 - `derive.comp_rsp` (*cumulative*)<br>    Corresponds to HAProxy's `comp_rsp` metric -  Number of HTTP responses that were compressed
 - `derive.compress_bps_in` (*cumulative*)<br>    Corresponds to HAProxy's `CompressBpsIn` metric.
 - `derive.compress_bps_out` (*cumulative*)<br>    Corresponds to HAProxy's `CompressBpsOut` metric.
 - `derive.connections` (*cumulative*)<br>    Corresponds to HAProxy's `CumConns` metric. Cumulative number of connections.
 - ***`derive.denied_request`*** (*cumulative*)<br>    Corresponds to HAProxy's `dreq` metric -  Requests denied because of security concerns. - For tcp this is because of a matched tcp-request content rule.
 - ***`derive.denied_response`*** (*cumulative*)<br>    Corresponds to HAProxy's `dresp` metric -  Responses denied because of security concerns. - For http this is because of a matched http-request rule, or
 - `derive.downtime` (*cumulative*)<br>    Corresponds to HAProxy's `downtime` metric -  Total downtime (in seconds). The value for the backend is the downtime for the whole backend, not the sum of the server downtime.
 - ***`derive.error_connection`*** (*cumulative*)<br>    Corresponds to HAProxy's `econ` metric -  Number of requests that encountered an error trying to connect to a backend server. The backend stat is the sum of the stat
 - ***`derive.error_request`*** (*cumulative*)<br>    Corresponds to HAProxy's `ereq` metric -  Request errors.
 - ***`derive.error_response`*** (*cumulative*)<br>    Corresponds to HAProxy's `eresp` metric -  Response errors. srv_abrt will be counted here also. Responses denied because of security concerns.
 - `derive.failed_checks` (*cumulative*)<br>    Corresponds to HAProxy's `chkfail` metric -  Number of failed checks. (Only counts checks failed when the server is up.)
 - ***`derive.redispatched`*** (*cumulative*)<br>    Corresponds to HAProxy's `wredis` metric -  Number of times a request was redispatched to another server. The server value counts the number of times that server was
 - `derive.request_total` (*cumulative*)<br>    Corresponds to HAProxy's `req_tot` metric -  Total number of HTTP requests received
 - ***`derive.requests`*** (*cumulative*)<br>    Corresponds to HAProxy's `CumReq` metric.
 - `derive.response_1xx` (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_1xx` metric -  Http responses with 1xx code
 - ***`derive.response_2xx`*** (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_2xx` metric -  Http responses with 2xx code
 - `derive.response_3xx` (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_3xx` metric -  Http responses with 3xx code
 - ***`derive.response_4xx`*** (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_4xx` metric -  Http responses with 4xx code
 - ***`derive.response_5xx`*** (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_5xx` metric -  Http responses with 5xx code
 - `derive.response_other` (*cumulative*)<br>    Corresponds to HAProxy's `hrsp_other` metric -  Http responses with other codes (protocol error)
 - ***`derive.retries`*** (*cumulative*)<br>    Corresponds to HAProxy's `wretr` metric -  Number of times a connection to a server was retried.
 - `derive.session_total` (*cumulative*)<br>    Corresponds to HAProxy's `stot` metric -  Cumulative number of connections
 - `derive.srv_abrt` (*cumulative*)<br>    Corresponds to HAProxy's `srv_abrt` metric -  Number of data transfers aborted by the server (inc. in eresp)
 - `derive.ssl_cache_lookups` (*cumulative*)<br>    Corresponds to HAProxy's `SslCacheLookups` metric.
 - `derive.ssl_cache_misses` (*cumulative*)<br>    Corresponds to HAProxy's `SslCacheMisses` metric.
 - `derive.ssl_connections` (*cumulative*)<br>    Corresponds to HAProxy's `CumSslConns` metric.
 - `derive.uptime_seconds` (*cumulative*)<br>    Corresponds to HAProxy's `Uptime_sec` metric.
 - `gauge.active_servers` (*gauge*)<br>    Number of active servers. This corresponds to HAProxy's "act" metric.
 - `gauge.backup_servers` (*gauge*)<br>    Number of backup servers. This corresponds to HAProxy's "bck" metric.
 - `gauge.check_duration` (*gauge*)<br>    Time in ms took to finish to last health check. This corresponds to HAProxy's "check_duration" metric.
 - ***`gauge.connection_rate`*** (*gauge*)<br>    Number of connections over the last elapsed second (frontend). This corresponds to HAProxy's "conn_rate" metric.
 - `gauge.connection_rate_max` (*gauge*)<br>    Highest known connection rate. This corresponds to HAProxy's "conn_rate_max" metric.
 - `gauge.current_connections` (*gauge*)<br>    Current number of connections. Corresponds to HAProxy's `CurrConns` metric.
 - `gauge.current_ssl_connections` (*gauge*)<br>    Corresponds to HAProxy's `CurrSslConns` metric.
 - `gauge.denied_tcp_connections` (*gauge*)<br>    Requests denied by 'tcp-request connection' rules. This corresponds to HAProxy's "dcon" metric.
 - `gauge.denied_tcp_sessions` (*gauge*)<br>    Requests denied by 'tcp-request session' rules. This corresponds to HAProxy's "dses" metric.
 - ***`gauge.idle_pct`*** (*gauge*)<br>    Corresponds to HAProxy's "Idle_pct" metric. Ratio of system polling time versus total time.
 - `gauge.intercepted_requests` (*gauge*)<br>    Cumulative number of intercepted requests, corresponds to HAProxys metric 'intercepted'
 - `gauge.last_session` (*gauge*)<br>    Number of seconds since last session was assigned to server/backend. This corresponds to HAProxy's "lastsess" metric.
 - `gauge.max_connection_rate` (*gauge*)<br>    Corresponds to HAProxy's `MaxConnRate` metric.
 - `gauge.max_connections` (*gauge*)<br>    Corresponds to HAProxy's `MaxConn` metric.
 - `gauge.max_pipes` (*gauge*)<br>    Corresponds to HAProxy's `MaxPipes` metric.
 - `gauge.max_session_rate` (*gauge*)<br>    Corresponds to HAProxy's `MaxSessRate` metric.
 - `gauge.max_ssl_connections` (*gauge*)<br>    Corresponds to HAProxy's `MaxSslConns` metric.
 - `gauge.pipes_free` (*gauge*)<br>    Corresponds to HAProxy's `PipesFree` metric.
 - `gauge.pipes_used` (*gauge*)<br>    Corresponds to HAProxy's `PipesUsed` metric.
 - ***`gauge.queue_current`*** (*gauge*)<br>    Corresponds to HAProxy's `qcur` metric -  Current queued requests. For the backend this reports the number queued without a server assigned.
 - `gauge.queue_limit` (*gauge*)<br>    Configured max queue for the server, 0 being no limit. Corresponds to HAProxy's "qlimit" metric.
 - `gauge.queue_max` (*gauge*)<br>    Max number of queued requests, queue_current, corresponds to HAProxy's 'qmax' metric.
 - ***`gauge.queue_time_avg`*** (*gauge*)<br>
 - ***`gauge.request_rate`*** (*gauge*)<br>    Corresponds to HAProxy's `req_rate` metric -  HTTP requests per second over last elapsed second
 - `gauge.request_rate_max` (*gauge*)<br>    Max number of HTTP requests per second observed. Corresponds to HAProxy's "req_rate_max" metric.
 - ***`gauge.response_time_avg`*** (*gauge*)<br>    Average total session time in ms over the last 1024 requests. Corresponds to HAProxy's "ttime" metric.
 - `gauge.run_queue` (*gauge*)<br>    Corresponds to HAProxy's `Run_queue` metric.
 - ***`gauge.session_current`*** (*gauge*)<br>    Corresponds to HAProxy's `scur` metric -  Current sessions
 - ***`gauge.session_rate`*** (*gauge*)<br>    Corresponds to HAProxy's `rate` metric -  Number of sessions per second over last elapsed second
 - ***`gauge.session_rate_all`*** (*gauge*)<br>
 - `gauge.session_rate_limit` (*gauge*)<br>    Configured limit on number of new sessions per second. Corresponds to HAProxy's "rate_lim" metric.
 - `gauge.session_rate_max` (*gauge*)<br>    Max number of new sessions per second
 - `gauge.session_time_average` (*gauge*)<br>    Average total session time in ms over the last 1024 requests. Corresponds to HAProxy's "ttime" metric.
 - ***`gauge.session_time_avg`*** (*gauge*)<br>
 - `gauge.ssl_backend_key_rate` (*gauge*)<br>    Corresponds to HAProxy's `SslBackendKeyRate` metric.
 - `gauge.ssl_frontend_key_rate` (*gauge*)<br>    Corresponds to HAProxy's `SslFrontendKeyRate` metric.
 - `gauge.ssl_rate` (*gauge*)<br>    Corresponds to HAProxy's `SslRate` metric.
 - `gauge.tasks` (*gauge*)<br>    Corresponds to HAProxy's `Tasks` metric.
 - `gauge.throttle` (*gauge*)<br>    Corresponds to HAProxy's `throttle` metric -  Current throttle percentage for the server, when slowstart is active, or no value if not in slowstart.
 - `gauge.zlib_mem_usage` (*gauge*)<br>    Corresponds to HAProxy's `ZlibMemUsage` metric.

### Non-default metrics (version 4.7.0+)

**The following information applies to the agent version 4.7.0+ that has
`enableBuiltInFiltering: true` set on the top level of the agent config.**

To emit metrics that are not _default_, you can add those metrics in the
generic monitor-level `extraMetrics` config option.  Metrics that are derived
from specific configuration options that do not appear in the above list of
metrics do not need to be added to `extraMetrics`.

To see a list of metrics that will be emitted you can run `agent-status
monitors` after configuring this monitor in a running agent instance.

### Legacy non-default metrics (version < 4.7.0)

**The following information only applies to agent version older than 4.7.0. If
you have a newer agent and have set `enableBuiltInFiltering: true` at the top
level of your agent config, see the section above. See upgrade instructions in
[Old-style whitelist filtering](../legacy-filtering.html#old-style-whitelist-filtering).**

If you have a reference to the `whitelist.json` in your agent's top-level
`metricsToExclude` config option, and you want to emit metrics that are not in
that whitelist, then you need to add an item to the top-level
`metricsToInclude` config option to override that whitelist (see [Inclusion
filtering](../legacy-filtering.html#inclusion-filtering).  Or you can just
copy the whitelist.json, modify it, and reference that in `metricsToExclude`.


