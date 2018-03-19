from http.server import HTTPServer, BaseHTTPRequestHandler
from collections import defaultdict
from contextlib import contextmanager
import gzip
import json
from google.protobuf import json_format
import signal
import threading
from time import time

from signalfx.generated_protocol_buffers \
    import signal_fx_protocol_buffers_pb2 as sf_pbuf

# This module collects metrics from the agent and can echo them back out for
# making assertions on the collected metrics.

# Fake the /v2/datapoint endpoint and just stick all of the metrics in a
# list
def _make_fake_ingest(datapoints, events):
    class FakeIngest(BaseHTTPRequestHandler):
        def do_POST(self):
            print("INGEST POST: %s" % self.path)
            body = self.rfile.read(int(self.headers.get('Content-Length')))
            is_json = "application/json" in self.headers.get("Content-Type")

            if "gzip" in self.headers.get("Content-Encoding", ""):
                body = gzip.decompress(body)

            if 'datapoint' in self.path:
                dp_upload = sf_pbuf.DataPointUploadMessage()
                if is_json:
                    json_format.Parse(body, dp_upload)
                else:
                    dp_upload.ParseFromString(body)
                datapoints.extend(dp_upload.datapoints)
            elif 'event' in self.path:
                event_upload = sf_pbuf.EventUploadMessage()
                if is_json:
                    json_format.Parse(body, event_upload)
                else:
                    event_upload.ParseFromString(body)
                events.extend(event_upload.events)
            else:
                self.send_response(404)
                self.end_headers()
                return

            self.send_response(200)
            self.send_header("Content-Type", "text/ascii")
            self.send_header("Content-Length", "4")
            self.end_headers()
            self.wfile.write("\"OK\"".encode("utf-8"))

    return HTTPServer(('127.0.0.1', 0), FakeIngest)


# Fake the dimension PUT method to capture dimension property/tag updates.
def _make_fake_api(dims):
    class FakeAPIServer(BaseHTTPRequestHandler):
        def do_PUT(self):
            if '/dimension/' not in self.path:
                self.send_response(404)
                self.end_headers()
                return

            body = self.rfile.read(int(self.headers.getheader('Content-Length')))

            dims[key][value] = json.loads(body)

            self.send_response(200)
            self.send_header("Content-Type", "text/ascii")
            self.send_header("Content-Length", "0")
            self.end_headers()

    return HTTPServer(('127.0.0.1', 0), FakeAPIServer)

# Starts up a new set of backend services that will run on a random port.  The
# returned object will have properties on it for datapoints, events, and dims.
# The fake servers will be stopped once the context manager block is exited.
@contextmanager
def start():
    # Data structures are thread-safe due to the GIL
    _datapoints = []
    _events = []
    _dims = defaultdict(defaultdict)

    ingest_httpd = _make_fake_ingest(_datapoints, _events)
    api_httpd = _make_fake_api(_dims)

    threading.Thread(target=ingest_httpd.serve_forever).start()
    threading.Thread(target=api_httpd.serve_forever).start()

    class FakeBackend:
        ingest_host = ingest_httpd.server_address[0]
        ingest_port = ingest_httpd.server_address[1]
        ingest_url = "http://%s:%d" % ingest_httpd.server_address

        api_host = api_httpd.server_address[0]
        api_port = api_httpd.server_address[1]
        api_url = "http://%s:%d" % api_httpd.server_address

        datapoints = _datapoints
        events = _events
        dims = _dims

    try:
        yield FakeBackend()
    finally:
        ingest_httpd.shutdown()
        api_httpd.shutdown()

