"""
Assertions about requests/data received on the fake backend
"""
import os
import re
import socket
import urllib.request
from base64 import b64encode
from collections import defaultdict
from http.client import HTTPException

import psutil
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf


def has_datapoint_with_metric_name(fake_services, metric_name):
    if hasattr(metric_name, "match"):
        return any(metric_name.match(dp.metric) for dp in fake_services.datapoints)
    return any(dp.metric == metric_name for dp in fake_services.datapoints)


def has_datapoint_with_dim_key(fake_services, dim_key):
    for dp in fake_services.datapoints:
        for dim in dp.dimensions:
            if dim.key == dim_key:
                return True
    return False


def has_all_dims(dp_or_event, dims):
    """
    Tests if `dims`'s are all in a certain datapoint or event
    """
    return dims.items() <= {d.key: d.value for d in dp_or_event.dimensions}.items()


# Tests if any datapoints has all of the given dimensions
def has_datapoint_with_all_dims(fake_services, dims):
    for dp in fake_services.datapoints:
        if has_all_dims(dp, dims):
            return True
    return False


# Tests if any datapoint received has the given dim key/value on it.
def has_datapoint_with_dim(fake_services, key, value):
    return has_datapoint_with_all_dims(fake_services, {key: value})


# Tests if all datapoints received have some or all the given dims.
def datapoints_have_some_or_all_dims(fake_services, dims):
    for dp in fake_services.datapoints:
        has_dim = False
        for dp_dim in dp.dimensions:
            if dims.get(dp_dim.key) == dp_dim.value:
                has_dim = True
                break
        if not has_dim:
            return False
    return True


def has_no_datapoint(fake_services, metric_name=None, dimensions=None, value=None, metric_type=None):
    """
    Returns True is there are no datapoints matching the given parameters
    """
    return not has_datapoint(fake_services, metric_name, dimensions, value, metric_type, count=1)


def has_datapoint(fake_services, metric_name=None, dimensions=None, value=None, metric_type=None, count=1):
    """
    Returns True if there is a datapoint seen in the fake_services backend that
    has the given attributes.  If a property is not specified it will not be
    considered.  Dimensions, if provided, will be tested as a subset of total
    set of dimensions on the datapoint and not the complete set.
    """
    found = 0
    # Try and cull the number of datapoints that have to be searched since we
    # have to check each datapoint.
    if dimensions is not None:
        datapoints = []
        for k, v in dimensions.items():
            datapoints += fake_services.datapoints_by_dim[f"{k}:{v}"]
    elif metric_name is not None:
        datapoints = fake_services.datapoints_by_metric[metric_name]
    else:
        datapoints = fake_services.datapoints

    for dp in fake_services.datapoints:
        if metric_name and dp.metric != metric_name:
            continue
        if dimensions and not has_all_dims(dp, dimensions):
            continue
        if metric_type and dp.metricType != metric_type:
            continue
        if value is not None:
            if dp.value.HasField("intValue"):
                if dp.value.intValue != value:
                    continue
            elif dp.value.HasField("doubleValue"):
                if dp.value.doubleValue != value:
                    continue
            else:
                # Non-numeric values aren't supported, so they always fail to
                # match
                continue
        found += 1
        if found >= count:
            return True
    return False


def tsid_for_datapoint(dp: sf_pbuf.DataPoint):
    key = f"{dp.metric}"
    dim_keys = []
    for dim in dp.dimensions:
        dim_keys.append(f"#{dim.key}|{dim.value}")

    dim_keys.sort()
    tsid = hash(key + "|".join(dim_keys))
    return tsid


def all_timeseries(fake_services):
    mts_by_id = defaultdict(list)
    for dp in fake_services.datapoints:
        mts_by_id[tsid_for_datapoint(dp)].append(dp)

    return dict(mts_by_id)


def has_time_series(fake_services, metric_name: str, dimensions: dict) -> bool:
    dp = sf_pbuf.DataPoint()
    dp.metric = metric_name
    for k, v in dimensions.items():
        dim = sf_pbuf.Dimension()
        dim.key = k
        dim.value = v
        dp.dimensions.extend([dim])  # pylint:disable=no-member

    tsid = tsid_for_datapoint(dp)
    return tsid in all_timeseries(fake_services)


def has_event_type(fake_services, event_type):
    for evt in fake_services.events:
        if evt.eventType == event_type:
            return True
    return False


# Tests if any event received has the given dim key/value on it.
def has_event_with_dim(fake_services, key, value):
    """
    Tests if any event received has the given dim key/value on it.
    """
    for event in fake_services.events:
        if has_all_dims(event, {key: value}):
            return True
    return False


def container_cmd_exit_0(container, command):
    """
    Tests if a command run against a container returns with an exit code of 0
    """
    code, _ = container.exec_run(command)
    return code == 0


def text_is_in_stream(stream, text):
    """
    Checks if the given text exists in a larger block of text, while ignoring
    line breaks.
    """
    return text.encode("utf-8") in b"".join(stream.readlines())


def has_log_message(output, level="info", message=""):
    """
    Returns True if the given message occurs in the output text at the given
    log level.
    """
    for line in output.splitlines():
        match = re.search(r"(?<=level=)\w+", line)
        if match is None:
            continue
        if level == match.group(0) and message in line:
            return True
    return False


def regex_search_matches_output(get_output, search):
    """
    Applies a regex search func to the current output
    """
    return search(get_output())


def udp_port_open_locally(port):
    """
    Returns true is the given port # is open on the local host
    """
    return os.system("cat /proc/net/udp /proc/net/udp6 | grep %s" % (hex(port)[2:].upper(),)) == 0


def local_tcp_port_has_connection(port):
    """
    Returns true is the given port # has an active connection to it
    """
    for conn in psutil.net_connections("tcp"):
        if conn.status != psutil.CONN_ESTABLISHED:
            continue
        if conn.raddr and conn.raddr.port == port:
            return True
    return False


def tcp_port_open_locally(port):
    """
    Returns True if the given TCP port is open on the local machine
    """
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    result = sock.connect_ex(("127.0.0.1", port))
    return result == 0


def any_metric_found(fake_services, metrics):
    """
    Check if any metric in `metrics` exist
    """
    return any([has_datapoint_with_metric_name(fake_services, m) for m in metrics])


def any_dim_key_found(fake_services, dim_keys):
    """
    Check if any dimension key in `dim_keys` exist
    """
    return any([has_datapoint_with_dim_key(fake_services, k) for k in dim_keys])


def any_metric_has_any_dim_key(fake_services, metrics, dim_keys):
    """
    Check if any metric in `metrics` with any dimension key in `dim_keys` exist
    """
    for dp in fake_services.datapoints:
        if dp.metric in metrics:
            for dim in dp.dimensions:
                if dim.key in dim_keys:
                    print('Found metric "%s" with dimension key "%s".' % (dp.metric, dim.key))
                    return True
    return False


def has_any_metric_or_dim(fake_services, metrics, dims):
    """
    Returns True if any of the given metrics or dims are present in the fake backend
    """
    if metrics and dims:
        return any_metric_has_any_dim_key(fake_services, metrics, dims)
    if metrics:
        return any_metric_found(fake_services, metrics)
    if dims:
        return any_dim_key_found(fake_services, dims)
    return False


def tcp_socket_open(host, port):
    """
    Returns True if there is an open TCP socket at the given host/port
    """
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(1)
    try:
        return sock.connect_ex((host, port)) == 0
    except socket.timeout:
        return False


def http_status(url=None, status=None, username=None, password=None, timeout=1, **kwargs):
    """
    Wrapper around urllib.request.urlopen() that returns True if
    the request returns the any of the specified HTTP status codes.  Accepts
    username and password keyword arguments for basic authorization.
    """
    if status is None:
        status = []

    try:
        # urllib expects url argument to either be a string url or a request object
        req = url if isinstance(url, urllib.request.Request) else urllib.request.Request(url)

        if username and password:
            # create basic authorization header
            auth = b64encode("{0}:{1}".format(username, password).encode("ascii")).decode("utf-8")
            req.add_header("Authorization", "Basic {0}".format(auth))

        return urllib.request.urlopen(req, timeout=timeout, **kwargs).getcode() in status
    except urllib.error.HTTPError as err:
        # urllib raises exceptions for some http error statuses
        return err.code in status
    except (urllib.error.URLError, socket.timeout, HTTPException, ConnectionResetError, ConnectionError):
        return False


def has_trace_span(  # pylint: disable=too-many-arguments
    fake_services, trace_id=None, span_id=None, parent_id=None, name=None, local_service_name=None, kind=None, tags=None
):
    """
    Returns True if there is a trace span seen in the fake_services backend that
    has the given attributes.  If a property is not specified it will not be
    considered.  `tags`, if provided, will be tested as a subset of total
    set of tags on the span and not the complete set.
    """
    for span in fake_services.spans:
        if trace_id and span.get("traceId") != trace_id:
            continue
        if span_id and span.get("id") != span_id:
            continue
        if parent_id and span.get("parentId") != parent_id:
            continue
        if name and span.get("name") != name:
            continue
        if kind and span.get("kind") != kind:
            continue
        if local_service_name and span.get("localEndpoint", {}).get("serviceName") != local_service_name:
            continue
        if tags and not has_all_tags(span, tags):
            continue
        return True
    return False


def has_all_tags(span, tags):
    """
    Tests if `tags`'s are all in a certain trace span
    """
    return tags.items() <= span.get("tags").items()


def all_datapoints_have_metric_name(fake_services, metric_name):
    if hasattr(metric_name, "match"):
        return all(metric_name.match(dp.metric) for dp in fake_services.datapoints)
    return all(dp.metric == metric_name for dp in fake_services.datapoints)


def all_datapoints_have_dims(fake_services, dims):
    return all(has_all_dims(dp, dims) for dp in fake_services.datapoints)


def all_datapoints_have_dim_key(fake_services, dim_key):
    if not fake_services.datapoints:
        return False
    for dp in fake_services.datapoints:
        if not has_dim_key(dp, dim_key):
            return False
    return True


def has_dim_key(dp_or_event, key):
    return any(dim.key == key for dim in dp_or_event.dimensions)


def all_datapoints_have_metric_name_and_dims(fake_services, metric_name, dims):
    return all_datapoints_have_metric_name(fake_services, metric_name) and all_datapoints_have_dims(fake_services, dims)


def all_datapoints_have_metric_name_and_dim_key(fake_services, metric_name, dim_key):
    return all_datapoints_have_metric_name(fake_services, metric_name) and all_datapoints_have_dim_key(
        fake_services, dim_key
    )


def has_dim_tag(fake_services, dim_name, dim_value, tag_value):
    """
    Tests if the given dimension has a certain tag
    """
    dim = fake_services.dims[dim_name][dim_value]
    tags = dim.get("tags", [])
    if tags is not None:
        return tag_value in tags
    return False


def has_dim_prop(fake_services, dim_name, dim_value, prop_name, prop_value=None):
    """
    Tests if the given dimension has a property.  If prop_value is None, tests
    for the presence of the property regardless of value.
    """
    dim = fake_services.dims[dim_name].get(dim_value, {})
    props = dim.get("customProperties", {})
    if props is not None:
        if prop_value is None:
            return prop_name in props
        return props.get(prop_name) == prop_value
    return False


def has_all_dim_props(fake_services, dim_name, dim_value, props):
    """
    Returns True if all of the `props` are in the given dimension.  Returns
    False if any are missing.  There can be additional properties on the
    dimension not covered by props.
    """
    for k, v in props.items():
        if not has_dim_prop(fake_services, dim_name=dim_name, dim_value=dim_value, prop_name=k, prop_value=v):
            return False
    return True
