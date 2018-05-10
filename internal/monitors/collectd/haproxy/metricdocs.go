package haproxy

// COUNTER(counter.connection_total): Cumulative number of connections (frontend). This corresponds to HAProxy's "conn_tot" metric.

// COUNTER(counter.server_selected_total): Number of times a server was selected, either for new sessions or when re-dispatching. This corresponds to HAProxy's "lbtot" metric.

// CUMULATIVE(derive.bytes_in): Corresponds to HAProxy's `bin` metric -  Bytes in

// CUMULATIVE(derive.bytes_out): Corresponds to HAProxy's `bout` metric -  Bytes out

// CUMULATIVE(derive.cli_abrt): Corresponds to HAProxy's `cli_abrt` metric -  Number of data transfers aborted by the client

// CUMULATIVE(derive.comp_byp): Corresponds to HAProxy's `comp_byp` metric -  Number of bytes that bypassed the HTTP compressor (CPU/BW limit)

// CUMULATIVE(derive.comp_in): Corresponds to HAProxy's `comp_in` metric -  Number of HTTP response bytes fed to the compressor

// CUMULATIVE(derive.comp_out): Corresponds to HAProxy's `comp_out` metric -  Number of HTTP response bytes emitted by the compressor

// CUMULATIVE(derive.comp_rsp): Corresponds to HAProxy's `comp_rsp` metric -  Number of HTTP responses that were compressed

// CUMULATIVE(derive.compress_bps_in): Corresponds to HAProxy's `CompressBpsIn` metric.

// CUMULATIVE(derive.compress_bps_out): Corresponds to HAProxy's `CompressBpsOut` metric.

// CUMULATIVE(derive.connections): Corresponds to HAProxy's `CumConns` metric. Cumulative number of connections.

// CUMULATIVE(derive.denied_request): Corresponds to HAProxy's `dreq` metric -  Requests denied because of security concerns. - For tcp this is because of a matched tcp-request content rule.

// CUMULATIVE(derive.denied_response): Corresponds to HAProxy's `dresp` metric -  Responses denied because of security concerns. - For http this is because of a matched http-request rule, or

// CUMULATIVE(derive.downtime): Corresponds to HAProxy's `downtime` metric -  Total downtime (in seconds). The value for the backend is the downtime for the whole backend, not the sum of the server downtime.

// CUMULATIVE(derive.error_connection): Corresponds to HAProxy's `econ` metric -  Number of requests that encountered an error trying to connect to a backend server. The backend stat is the sum of the stat

// CUMULATIVE(derive.error_request): Corresponds to HAProxy's `ereq` metric -  Request errors.

// CUMULATIVE(derive.error_response): Corresponds to HAProxy's `eresp` metric -  Response errors. srv_abrt will be counted here also. Responses denied because of security concerns.

// CUMULATIVE(derive.failed_checks): Corresponds to HAProxy's `chkfail` metric -  Number of failed checks. (Only counts checks failed when the server is up.)

// CUMULATIVE(derive.redispatched): Corresponds to HAProxy's `wredis` metric -  Number of times a request was redispatched to another server. The server value counts the number of times that server was

// CUMULATIVE(derive.request_total): Corresponds to HAProxy's `req_tot` metric -  Total number of HTTP requests received

// CUMULATIVE(derive.requests): Corresponds to HAProxy's `CumReq` metric.

// CUMULATIVE(derive.response_1xx): Corresponds to HAProxy's `hrsp_1xx` metric -  Http responses with 1xx code

// CUMULATIVE(derive.response_2xx): Corresponds to HAProxy's `hrsp_2xx` metric -  Http responses with 2xx code

// CUMULATIVE(derive.response_3xx): Corresponds to HAProxy's `hrsp_3xx` metric -  Http responses with 3xx code

// CUMULATIVE(derive.response_4xx): Corresponds to HAProxy's `hrsp_4xx` metric -  Http responses with 4xx code

// CUMULATIVE(derive.response_5xx): Corresponds to HAProxy's `hrsp_5xx` metric -  Http responses with 5xx code

// CUMULATIVE(derive.response_other): Corresponds to HAProxy's `hrsp_other` metric -  Http responses with other codes (protocol error)

// CUMULATIVE(derive.retries): Corresponds to HAProxy's `wretr` metric -  Number of times a connection to a server was retried.

// CUMULATIVE(derive.session_total): Corresponds to HAProxy's `stot` metric -  Cumulative number of connections

// CUMULATIVE(derive.srv_abrt): Corresponds to HAProxy's `srv_abrt` metric -  Number of data transfers aborted by the server (inc. in eresp)

// CUMULATIVE(derive.ssl_cache_lookups): Corresponds to HAProxy's `SslCacheLookups` metric.

// CUMULATIVE(derive.ssl_cache_misses): Corresponds to HAProxy's `SslCacheMisses` metric.

// CUMULATIVE(derive.ssl_connections): Corresponds to HAProxy's `CumSslConns` metric.

// CUMULATIVE(derive.uptime_seconds): Corresponds to HAProxy's `Uptime_sec` metric.

// GAUGE(gauge.active_servers): Number of active servers. This corresponds to HAProxy's "act" metric.

// GAUGE(gauge.backup_servers): Number of backup servers. This corresponds to HAProxy's "bck" metric.

// GAUGE(gauge.check_duration): Time in ms took to finish to last health check. This corresponds to HAProxy's "check_duration" metric.

// GAUGE(gauge.connection_rate): Number of connections over the last elapsed second (frontend). This corresponds to HAProxy's "conn_rate" metric.

// GAUGE(gauge.connection_rate_max): Highest known connection rate. This corresponds to HAProxy's "conn_rate_max" metric.

// GAUGE(gauge.current_connections): Current number of connections. Corresponds to HAProxy's `CurrConns` metric.

// GAUGE(gauge.current_ssl_connections): Corresponds to HAProxy's `CurrSslConns` metric.

// GAUGE(gauge.denied_tcp_connections): Requests denied by 'tcp-request connection' rules. This corresponds to HAProxy's "dcon" metric.

// GAUGE(gauge.denied_tcp_sessions): Requests denied by 'tcp-request session' rules. This corresponds to HAProxy's "dses" metric.

// GAUGE(gauge.idle_pct): Corresponds to HAProxy's "Idle_pct" metric. Ratio of system polling time versus total time.

// GAUGE(gauge.intercepted_requests): Cumulative number of intercepted requests, corresponds to HAProxys metric 'intercepted'

// GAUGE(gauge.last_session): Number of seconds since last session was assigned to server/backend. This corresponds to HAProxy's "lastsess" metric.

// GAUGE(gauge.max_connection_rate): Corresponds to HAProxy's `MaxConnRate` metric.

// GAUGE(gauge.max_connections): Corresponds to HAProxy's `MaxConn` metric.

// GAUGE(gauge.max_pipes): Corresponds to HAProxy's `MaxPipes` metric.

// GAUGE(gauge.max_session_rate): Corresponds to HAProxy's `MaxSessRate` metric.

// GAUGE(gauge.max_ssl_connections): Corresponds to HAProxy's `MaxSslConns` metric.

// GAUGE(gauge.pipes_free): Corresponds to HAProxy's `PipesFree` metric.

// GAUGE(gauge.pipes_used): Corresponds to HAProxy's `PipesUsed` metric.

// GAUGE(gauge.queue_current): Corresponds to HAProxy's `qcur` metric -  Current queued requests. For the backend this reports the number queued without a server assigned.

// GAUGE(gauge.queue_limit): Configured max queue for the server, 0 being no limit. Corresponds to HAProxy's "qlimit" metric.

// GAUGE(gauge.queue_max): Max number of queued requests, queue_current, corresponds to HAProxy's 'qmax' metric.

// GAUGE(gauge.request_rate): Corresponds to HAProxy's `req_rate` metric -  HTTP requests per second over last elapsed second

// GAUGE(gauge.request_rate_max): Max number of HTTP requests per second observed. Corresponds to HAProxy's "req_rate_max" metric.

// GAUGE(gauge.run_queue): Corresponds to HAProxy's `Run_queue` metric.

// GAUGE(gauge.session_current): Corresponds to HAProxy's `scur` metric -  Current sessions

// GAUGE(gauge.session_rate): Corresponds to HAProxy's `rate` metric -  Number of sessions per second over last elapsed second

// GAUGE(gauge.session_rate_limit): Configured limit on number of new sessions per second. Corresponds to HAProxy's "rate_lim" metric.

// GAUGE(gauge.session_rate_max): Max number of new sessions per second

// GAUGE(gauge.session_time_average): Average total session time in ms over the last 1024 requests. Corresponds to HAProxy's "ttime" metric.

// GAUGE(gauge.ssl_backend_key_rate): Corresponds to HAProxy's `SslBackendKeyRate` metric.

// GAUGE(gauge.ssl_frontend_key_rate): Corresponds to HAProxy's `SslFrontendKeyRate` metric.

// GAUGE(gauge.ssl_rate): Corresponds to HAProxy's `SslRate` metric.

// GAUGE(gauge.tasks): Corresponds to HAProxy's `Tasks` metric.

// GAUGE(gauge.throttle): Corresponds to HAProxy's `throttle` metric -  Current throttle percentage for the server, when slowstart is active, or no value if not in slowstart.

// GAUGE(gauge.zlib_mem_usage): Corresponds to HAProxy's `ZlibMemUsage` metric.

