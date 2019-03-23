// DO NOT EDIT. This file is auto-generated.

package prometheusnginxvts

const (
	nginxVtsInfo                            = "nginx_vts_info"
	nginxVtsStartTimeSeconds                = "nginx_vts_start_time_seconds"
	nginxVtsMainConnections                 = "nginx_vts_main_connections"
	nginxVtsMainShmUsageBytes               = "nginx_vts_main_shm_usage_bytes"
	nginxVtsServerBytesTotal                = "nginx_vts_server_bytes_total"
	nginxVtsServerRequestsTotal             = "nginx_vts_server_requests_total"
	nginxVtsServerRequestSecondsTotal       = "nginx_vts_server_request_seconds_total"
	nginxVtsServerCacheTotal                = "nginx_vts_server_cache_total"
	nginxVtsServerRequestSeconds            = "nginx_vts_server_request_seconds"
	nginxVtsServerRequestDurationSeconds    = "nginx_vts_server_request_duration_seconds"
	nginxVtsUpstreamBytesTotal              = "nginx_vts_upstream_bytes_total"
	nginxVtsUpstreamRequestsTotal           = "nginx_vts_upstream_requests_total"
	nginxVtsUpstreamRequestSecondsTotal     = "nginx_vts_upstream_request_seconds_total"
	nginxVtsUpstreamRequestSeconds          = "nginx_vts_upstream_request_seconds"
	nginxVtsUpstreamResponseSecondsTotal    = "nginx_vts_upstream_response_seconds_total"
	nginxVtsUpstreamResponseSeconds         = "nginx_vts_upstream_response_seconds"
	nginxVtsUpstreamRequestDurationSeconds  = "nginx_vts_upstream_request_duration_seconds"
	nginxVtsUpstreamResponseDurationSeconds = "nginx_vts_upstream_response_duration_seconds"
)

var includedMetrics = map[string]bool{
	nginxVtsMainConnections:        true,
	nginxVtsServerRequestsTotal:    true,
	nginxVtsServerRequestSeconds:   true,
	nginxVtsUpstreamRequestSeconds: true,
}
