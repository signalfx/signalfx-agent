import asyncio
import gzip
import json
import random
import socket
import string
import sys
import threading
from collections import OrderedDict, defaultdict
from contextlib import contextmanager
from queue import Queue

from google.protobuf import json_format
from sanic import Sanic, response
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf

# This module collects metrics from the agent and can echo them back out for
# making assertions on the collected metrics.
from tests.helpers.formatting import get_metric_type

STOP = type("STOP", (), {})


def bind_tcp_socket(host="127.0.0.1", port=0):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind((host, port))

    return (sock, sock.getsockname()[1])


# Fake the /v2/datapoint endpoint and just stick all of the metrics in a
# list
# pylint: disable=unused-variable
def _make_fake_ingest(datapoint_queue, events, spans, save_datapoints, save_events, save_spans):
    app = Sanic(name="".join(random.choices(string.ascii_lowercase, k=16)))

    @app.middleware("request")
    async def compress_request(request):
        if "Content-Encoding" in request.headers:
            if "gzip" in request.headers["Content-Encoding"]:
                request.body = gzip.decompress(request.body)

    @app.post("/v2/datapoint")
    async def handle_datapoints(request):
        is_json = "application/json" in request.headers.get("content-type")

        dp_upload = sf_pbuf.DataPointUploadMessage()
        if is_json:
            json_format.Parse(request.body, dp_upload)
        else:
            dp_upload.ParseFromString(request.body)
        if save_datapoints:
            datapoint_queue.put(dp_upload)
        return response.json("OK")

    @app.post("/v2/event")
    async def handle_event(request):
        is_json = "application/json" in request.headers.get("content-type")

        event_upload = sf_pbuf.EventUploadMessage()
        if is_json:
            json_format.Parse(request.body, event_upload)
        else:
            event_upload.ParseFromString(request.body)
        if save_events:
            events.extend(event_upload.events)  # pylint: disable=no-member
        return response.json("OK")

    @app.post("/v1/trace")
    async def handle_trace(request):
        if save_spans:
            spans.extend(request.json)
        return response.json("OK")

    return app


# Fake the dimension PUT method to capture dimension property/tag updates.
# pylint: disable=unused-variable
def _add_fake_dimension_api(app, dims):
    @app.get("/v2/dimension/<key>/<value>")
    async def get_dim(_, key, value):
        dim = dims.get(key, {}).get(value)
        if not dim:
            return response.json({}, status=404)
        return response.json(
            {"key": key, "value": value, "customProperties": dim.get("customProperties"), "tags": dim.get("tags")}
        )

    @app.put("/v2/dimension/<key>/<value>")
    async def put_dim(request, key, value):
        content = request.json
        dims[key][value] = content
        return response.json({})

    @app.patch("/v2/dimension/<key>/<value>/_/sfxagent")
    async def patch_dim(request, key, value):
        content = request.json

        # The API won't accept these on this endpoint so make sure they aren't
        # present
        assert content.get("key") is None
        assert content.get("value") is None

        content["key"] = key
        content["value"] = value

        prop_keys_to_delete = []
        props_to_add = content.get("customProperties", {})
        for k, v in props_to_add.items():
            if v is None:
                prop_keys_to_delete.append(k)

        for k in prop_keys_to_delete:
            del props_to_add[k]

        existing = dims[key].get(value)
        if not existing:
            dims[key][value] = {"customProperties": props_to_add, "tags": content.get("tags", [])}
            return response.json({})

        existing_props = existing.get("customProperties", {})
        existing_props.update(props_to_add)
        existing["customProperties"] = existing_props

        for k in prop_keys_to_delete:
            del existing_props[k]

        existing_tags = existing.get("tags", [])
        existing_tags.extend(content.get("tags", []))
        existing["tags"] = existing_tags

        for tag in content.get("tagsToRemove", []):
            existing_tags.remove(tag)

        return response.json(existing)

    return app


def _add_fake_correlation_api(app, dims, correlation_api_status_code):
    @app.put("/v2/apm/correlate/<key>/<value>/service")
    async def put_service(request, key, value):
        service = request.body.decode("utf-8")
        dim = dims.get(key, {}).get(value)
        if not dim:
            dims[key] = {value: {}}
            dims[key][value] = {"sf_services": [service]}
        elif key in ("kubernetes_pod_uid", "container_id"):
            dims[key][value]["sf_services"] = [service]
        else:
            dim_services = dim.get("sf_services")
            if not dim_services:
                dim_services = [service]
            else:
                dim_services.append(service)
            dims[key][value]["sf_services"] = dim_services
        return response.json({}, correlation_api_status_code)

    @app.put("/v2/apm/correlate/<key>/<value>/environment")
    async def put_environment(request, key, value):
        environment = request.body.decode("utf-8")
        dim = dims.get(key, {}).get(value)
        if not dim:
            dims[key] = {value: {}}
            dims[key][value] = {"sf_environments": [environment]}
        elif key in ("kubernetes_pod_uid", "container_id"):
            dims[key][value]["sf_environments"] = [environment]
        else:
            dim_environments = dim.get("sf_environments")
            if not dim_environments:
                dim_environments = [environment]
            else:
                dim_environments.append(environment)
            dims[key][value]["sf_environments"] = dim_environments
        return response.json({}, correlation_api_status_code)

    @app.delete("/v2/apm/correlate/<key>/<value>/service/<prop_value>")
    async def delete_service(_, key, value, prop_value):
        dim = dims.get(key, {}).get(value)
        if not dim:
            return response.json({})
        services = dim.get("sf_services")
        if prop_value in services:
            services.remove(prop_value)
        dim["sf_services"] = services
        return response.json({}, correlation_api_status_code)

    @app.delete("/v2/apm/correlate/<key>/<value>/environment/<prop_value>")
    async def delete_environment(_, key, value, prop_value):
        dim = dims.get(key, {}).get(value)
        if not dim:
            return response.json({})
        environments = dim.get("sf_environments")
        if prop_value in environments:
            environments.remove(prop_value)
        dim["sf_environments"] = environments
        return response.json({}, correlation_api_status_code)

    @app.get("/v2/apm/correlate/<key>/<value>")
    async def get_correlation(_, key, value):
        dim = dims.get(key, {}).get(value)
        if not dim:
            return response.json({})
        props = {}
        services = dim.get("sf_services")
        if services:
            props["sf_services"] = services
        environments = dim.get("sf_environments")
        if environments:
            props["sf_environments"] = environments
        return response.json(props, correlation_api_status_code)

    return app


def _make_fake_splunk_hec(entries):
    app = Sanic(name="".join(random.choices(string.ascii_lowercase, k=16)))

    @app.middleware("request")
    async def compress_request(request):
        if "Content-Encoding" in request.headers:
            if "gzip" in request.headers["Content-Encoding"]:
                request.body = gzip.decompress(request.body)

    @app.post("/services/collector")
    async def handle_entries(request):
        decoder = json.JSONDecoder()
        buffer = request.body.decode("utf-8")
        while buffer:
            result, index = decoder.raw_decode(buffer)
            entries.append(result)
            buffer = buffer[index:].lstrip()

        return response.json({})

    return app


# Starts up a new set of backend services that will run on a random port.  The
# returned object will have properties on it for datapoints, events, and dims.
# The fake servers will be stopped once the context manager block is exited.
# pylint: disable=too-many-locals,too-many-statements,too-many-arguments
@contextmanager
def start(
    ip_addr="127.0.0.1",
    ingest_port=0,
    api_port=0,
    splunk_hec_port=None,
    save_datapoints=True,
    save_events=True,
    save_spans=True,
    correlation_api_status_code=200,
):
    # Data structures are thread-safe due to the GIL
    _dp_upload_queue = Queue()
    _datapoints = []
    _datapoints_by_metric = defaultdict(list)
    _datapoints_by_dim = defaultdict(list)
    _events = []
    _spans = []
    _dims = defaultdict(defaultdict)
    _splunk_hec_entries = []

    ingest_app = _make_fake_ingest(_dp_upload_queue, _events, _spans, save_datapoints, save_events, save_spans)

    api_app = Sanic(name="".join(random.choices(string.ascii_lowercase, k=16)))
    _add_fake_dimension_api(api_app, _dims)
    _add_fake_correlation_api(api_app, _dims, correlation_api_status_code)

    [ingest_sock, _ingest_port] = bind_tcp_socket(ip_addr, ingest_port)
    [api_sock, _api_port] = bind_tcp_socket(ip_addr, api_port)

    ingest_loop = asyncio.new_event_loop()

    async def start_ingest_server():
        ingest_app.config.REQUEST_TIMEOUT = ingest_app.config.KEEP_ALIVE_TIMEOUT = 1000
        ingest_server = await ingest_app.create_server(sock=ingest_sock, access_log=False, return_asyncio_server=True)
        await ingest_server.startup()
        ingest_loop.create_task(ingest_server)

    ingest_loop.create_task(start_ingest_server())
    threading.Thread(target=ingest_loop.run_forever, daemon=True).start()

    api_loop = asyncio.new_event_loop()

    async def start_api_server():
        api_app.config.REQUEST_TIMEOUT = api_app.config.KEEP_ALIVE_TIMEOUT = 1000
        api_server = await api_app.create_server(sock=api_sock, access_log=False, return_asyncio_server=True)
        await api_server.startup()
        api_loop.create_task(api_server)

    api_loop.create_task(start_api_server())
    threading.Thread(target=api_loop.run_forever, daemon=True).start()

    splunk_hec_loop = asyncio.new_event_loop()

    if splunk_hec_port is not None:
        splunk_hec_app = _make_fake_splunk_hec(_splunk_hec_entries)
        [splunk_hec_sock, _splunk_hec_port] = bind_tcp_socket(ip_addr, splunk_hec_port)

        async def start_splunk_hec_server():
            splunk_hec_app.config.REQUEST_TIMEOUT = splunk_hec_app.config.KEEP_ALIVE_TIMEOUT = 1000
            splunk_hec_server = await splunk_hec_app.create_server(
                sock=splunk_hec_sock, access_log=False, return_asyncio_server=True
            )
            await splunk_hec_server.startup()
            splunk_hec_loop.create_task(splunk_hec_server)

        splunk_hec_loop.create_task(start_splunk_hec_server())
        threading.Thread(target=splunk_hec_loop.run_forever, daemon=True).start()

    def _add_datapoints():
        """
        This is an attempt at making the datapoint endpoint have more throughput for heavy load tests.
        """
        while True:
            dp_upload = _dp_upload_queue.get()
            if dp_upload is STOP:
                return
            _datapoints.extend(dp_upload.datapoints)  # pylint: disable=no-member
            for dp in dp_upload.datapoints:  # pylint: disable=no-member
                _datapoints_by_metric[dp.metric].append(dp)
                for dim in dp.dimensions:
                    _datapoints_by_dim[f"{dim.key}:{dim.value}"].append(dp)

    threading.Thread(target=_add_datapoints, daemon=True).start()

    class FakeBackend:  # pylint: disable=too-few-public-methods
        ingest_host = ip_addr
        ingest_port = _ingest_port
        ingest_url = f"http://{ingest_host}:{ingest_port}"

        api_host = ip_addr
        api_port = _api_port
        api_url = f"http://{api_host}:{api_port}"

        if splunk_hec_port is not None:
            splunk_hec_host = ip_addr
            splunk_hec_url = f"http://{splunk_hec_host}:{_splunk_hec_port}/services/collector"

        datapoints = _datapoints
        datapoints_by_metric = _datapoints_by_metric
        datapoints_by_dim = _datapoints_by_dim
        events = _events
        spans = _spans
        dims = _dims
        splunk_entries = _splunk_hec_entries

        def dump_json(self):
            out = OrderedDict()
            dps = [dp[0] for dp in self.datapoints_by_metric.values()]
            metrics = {(dp.metric, dp.metricType) for dp in dps}
            out["metrics"] = {metric: {"type": get_metric_type(metric_type)} for metric, metric_type in sorted(metrics)}
            out["dimensions"] = sorted(set(self.datapoints_by_dim))
            out["common_dimensions"] = []

            # Set dimensions that are present on all datapoints.
            for dim, dps in self.datapoints_by_dim.items():
                if len({dp.metric for dp in dps}) == len(metrics):
                    out["common_dimensions"].append(dim)

            json.dump(out, sys.stdout, indent=2)

        def reset_datapoints(self):
            self.datapoints.clear()
            self.datapoints_by_metric.clear()
            self.datapoints_by_dim.clear()

    try:
        yield FakeBackend()
    finally:
        ingest_sock.close()
        api_sock.close()
        api_loop.stop()
        ingest_loop.stop()

        _dp_upload_queue.put(STOP)

        if splunk_hec_port is not None:
            splunk_hec_loop.stop()
