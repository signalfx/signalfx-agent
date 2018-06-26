package kong

// CUMULATIVE(counter.kong.connections.accepted): Total number of all accepted connections. 
// CUMULATIVE(counter.kong.connections.handled): Total number of all handled connections (accounting for resource limits). 
// CUMULATIVE(counter.kong.kong.latency): Time spent in Kong request handling and balancer (ms). 
// CUMULATIVE(counter.kong.requests.count): Total number of all requests made to Kong API and proxy server. 
// CUMULATIVE(counter.kong.requests.latency): Time elapsed between the first bytes being read from each client request and the log writes after the last bytes were sent to the clients (ms). 
// CUMULATIVE(counter.kong.requests.size): Total bytes received/proxied from client requests. 
// CUMULATIVE(counter.kong.responses.count): Total number of responses provided to clients. 
// CUMULATIVE(counter.kong.responses.size): Total bytes sent/proxied to clients. 
// CUMULATIVE(counter.kong.upstream.latency): Time spent waiting for upstream response (ms). 
// GAUGE(gauge.kong.connections.active): The current number of active client connections (includes waiting). 
// GAUGE(gauge.kong.connections.reading): The current number of connections where nginx is reading the request header. 
// GAUGE(gauge.kong.connections.waiting): The current number of idle client connections waiting for a request. 
// GAUGE(gauge.kong.connections.writing): The current number of connections where nginx is writing the response back to the client. 
// GAUGE(gauge.kong.database.reachable): kong.dao:db.reachable() at time of metric query 
