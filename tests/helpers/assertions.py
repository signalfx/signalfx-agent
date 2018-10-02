"""
Assertions about requests/data received on the fake backend
"""
import os
import re
import socket
import urllib.request
from base64 import b64encode
from http.client import HTTPException


def has_datapoint_with_metric_name(fake_services, metric_name):
    for dp in fake_services.datapoints:
        if dp.metric == metric_name:
            return True
    return False


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


def has_datapoint(fake_services, metric_name=None, dimensions=None, value=None):
    """
    Returns True if there is a datapoint seen in the fake_services backend that
    has the given attributes.  If a property is not specified it will not be
    considered.  Dimensions, if provided, will be tested as a subset of total
    set of dimensions on the datapoint and not the complete set.
    """
    for dp in fake_services.datapoints:
        if metric_name and dp.metric != metric_name:
            continue
        if dimensions and not has_all_dims(dp, dimensions):
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
        return True
    return False


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
    return os.system("cat /proc/net/udp | grep %s" % (hex(port)[2:].upper(),)) == 0


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
