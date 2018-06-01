from base64 import b64encode
import os
import re
import socket
import urllib.request


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


# Tests if `dims`'s are all in a certain datapoint or event
def has_all_dims(dp_or_event, dims):
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


# Tests if any event received has the given dim key/value on it.
def has_event_with_dim(fake_services, key, value):
    for ev in fake_services.events:
        if has_all_dims(ev, {key: value}):
            return True
    return False


# Tests if a command run against a container returns with an exit code of 0
def container_cmd_exit_0(container, command):
    code, _ = container.exec_run(command)
    return code == 0


# This won't work very robustly if the text spans multiple lines.
def text_is_in_stream(stream, text):
    return text.encode("utf-8") in b"".join(stream.readlines())


def has_log_message(output, level="info", message=""):
    for l in output.splitlines():
        m = re.search(r'(?<=level=)\w+', l)
        if m is None:
            continue
        if level == m.group(0) and message in l:
            return True
    return False


def udp_port_open_locally(port):
    return os.system("cat /proc/net/udp | grep %s" % (hex(port)[2:].upper(),)) == 0


# check if any metric in `metrics` exist
def any_metric_found(fake_services, metrics):
    return any([has_datapoint_with_metric_name(fake_services, m) for m in metrics])


# check if any dimension key in `dim_keys` exist
def any_dim_key_found(fake_services, dim_keys):
    return any([has_datapoint_with_dim_key(fake_services, k) for k in dim_keys])


# check if any metric in `metrics` with any dimension key in `dim_keys` exist
def any_metric_has_any_dim_key(fake_services, metrics, dim_keys):
    for dp in fake_services.datapoints:
        if dp.metric in metrics:
            for dim in dp.dimensions:
                if dim.key in dim_keys:
                    print("Found metric \"%s\" with dimension key \"%s\"." % (dp.metric, dim.key))
                    return True
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


def http_status(url=None, status=[], username=None, password=None, timeout=1, *args, **kwargs):
    """
    Wrapper around urllib.request.urlopen() that returns True if
    the request returns the any of the specified HTTP status codes.  Accepts
    username and password keyword arguments for basic authorization.
    """
    try:
        # urllib expects url argument to either be a string url or a request object
        req = url if isinstance(url, urllib.request.Request) else urllib.request.Request(url)

        if username and password:
            # create basic authorization header
            auth = b64encode('{0}:{1}'.format(username, password).encode('ascii')).decode('utf-8')
            req.add_header('Authorization', 'Basic {0}'.format(auth))

        return urllib.request.urlopen(req, *args,
                                      timeout=timeout, **kwargs).getcode() in status
    except urllib.error.HTTPError as err:
        # urllib raises exceptions for some http error statuses
        return err.code in status
    except (urllib.error.URLError, socket.timeout):
        return False
