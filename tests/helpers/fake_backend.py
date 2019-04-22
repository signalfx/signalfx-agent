import asyncio
import gzip
import socket
import threading
from collections import defaultdict
from contextlib import contextmanager
from queue import Queue

from google.protobuf import json_format
from sanic import Sanic, response
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf

# This module collects metrics from the agent and can echo them back out for
# making assertions on the collected metrics.

STOP = type("STOP", (), {})


def free_tcp_socket(host="127.0.0.1"):
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind((host, 0))

    return (sock, sock.getsockname()[1])


# Fake the /v2/datapoint endpoint and just stick all of the metrics in a
# list
# pylint: disable=unused-variable
def _make_fake_ingest(datapoint_queue, events, spans):
    app = Sanic()

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
        events.extend(event_upload.events)  # pylint: disable=no-member
        return response.json("OK")

    @app.post("/v1/trace")
    async def handle_trace(request):
        spans.extend(request.json)
        return response.json([])

    return app


# Fake the dimension PUT method to capture dimension property/tag updates.
# pylint: disable=unused-variable
def _make_fake_api(dims):
    app = Sanic()

    @app.put("/v2/dimension/<key>/<value>")
    async def put_dim(request, key, value):
        content = request.json
        dims[key][value] = content
        return response.json({})

    return app


# Starts up a new set of backend services that will run on a random port.  The
# returned object will have properties on it for datapoints, events, and dims.
# The fake servers will be stopped once the context manager block is exited.
# pylint: disable=too-many-locals
@contextmanager
def start(ip_addr="127.0.0.1"):
    # Data structures are thread-safe due to the GIL
    _dp_upload_queue = Queue()
    _datapoints = []
    _datapoints_by_metric = defaultdict(list)
    _datapoints_by_dim = defaultdict(list)
    _events = []
    _spans = []
    _dims = defaultdict(defaultdict)

    ingest_app = _make_fake_ingest(_dp_upload_queue, _events, _spans)
    api_app = _make_fake_api(_dims)

    [ingest_sock, _ingest_port] = free_tcp_socket(ip_addr)
    [api_sock, _api_port] = free_tcp_socket(ip_addr)

    loop = asyncio.new_event_loop()

    async def start_servers():
        ingest_app.config.REQUEST_TIMEOUT = ingest_app.config.KEEP_ALIVE_TIMEOUT = 1000
        api_app.config.REQUEST_TIMEOUT = api_app.config.KEEP_ALIVE_TIMEOUT = 1000
        ingest_server = ingest_app.create_server(sock=ingest_sock, access_log=False)
        api_server = api_app.create_server(sock=api_sock, access_log=False)

        loop.create_task(ingest_server)
        loop.create_task(api_server)

    loop.create_task(start_servers())
    threading.Thread(target=loop.run_forever).start()

    def add_datapoints():
        while True:
            dp_upload = _dp_upload_queue.get()
            if dp_upload is STOP:
                return
            _datapoints.extend(dp_upload.datapoints)  # pylint: disable=no-member
            for dp in dp_upload.datapoints:  # pylint: disable=no-member
                _datapoints_by_metric[dp.metric].append(dp)
                for dim in dp.dimensions:
                    _datapoints_by_dim[f"{dim.key}:{dim.value}"].append(dp)

    threading.Thread(target=add_datapoints).start()

    class FakeBackend:  # pylint: disable=too-few-public-methods
        ingest_host = ip_addr
        ingest_port = _ingest_port
        ingest_url = f"http://{ingest_host}:{ingest_port}"

        api_host = ip_addr
        api_port = _api_port
        api_url = f"http://{api_host}:{api_port}"

        datapoints = _datapoints
        datapoints_by_metric = _datapoints_by_metric
        datapoints_by_dim = _datapoints_by_dim
        events = _events
        spans = _spans
        dims = _dims

        def reset_datapoints(self):
            self.datapoints.clear()
            self.datapoints_by_metric.clear()
            self.datapoints_by_dim.clear()

    try:
        yield FakeBackend()
    finally:
        ingest_sock.close()
        api_sock.close()
        loop.stop()
        _dp_upload_queue.put(STOP)
