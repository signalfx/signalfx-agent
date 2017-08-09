import json
import logging
import sys
from threading import Lock
import traceback
import zmq

from .monitors import Monitors

logger = logging.getLogger()

DATAPOINTS_TOPIC = "datapoints"

class NeoPy(object):
    def __init__(self, register_path, configure_path, shutdown_path, datapoints_path, logging_path):
        self.zmq_ctx = zmq.Context.instance()

        self.register_socket = self.zmq_ctx.socket(zmq.REP)
        self.register_path = register_path

        self.configure_socket = self.zmq_ctx.socket(zmq.REP)
        self.configure_path = configure_path

        self.shutdown_socket = self.zmq_ctx.socket(zmq.SUB)
        self.shutdown_path = shutdown_path

        self.datapoints_socket = self.zmq_ctx.socket(zmq.PUB)
        self.datapoints_lock = Lock()
        self.datapoints_path = datapoints_path

        self.logging_socket = self.zmq_ctx.socket(zmq.PUB)
        self.logging_path = logging_path

    def run(self):
        self.register_socket.connect(self.register_path)

        self.configure_socket.connect(self.configure_path)

        self.shutdown_socket.connect(self.shutdown_path)
        self.shutdown_socket.subscribe("shutdown")

        self.datapoints_socket.connect(self.datapoints_path)

        self.logging_socket.connect(self.logging_path)
        logger.addHandler(ZMQHandler(self.logging_socket))

        self.monitors = Monitors(self.send_datapoint)

        read_sockets = [
            self.register_socket,
            self.configure_socket,
            self.shutdown_socket
        ]

        while True:
            rlist, wlist, xlist = zmq.select(read_sockets, [], read_sockets)
            if len(xlist) > 0:
                logging.error("Socket had exception: %s" % xlist)

            self.process_messages(rlist)

    def process_messages(self, read_ready_sockets):
        for sock in read_ready_sockets:
            if sock == self.register_socket:
                # The request has no body
                sock.recv()
                monitor_names = self.monitors.registered_monitors

                logging.info("Registering monitors: %s" % monitor_names)

                self.register_socket.send_json({
                    "monitors": monitor_names
                })

            elif sock == self.configure_socket:
                try:
                    success = self.monitors.configure(sock.recv_json())
                    self.configure_socket.send_json({
                        "success": success,
                        "error": None
                    })
                except Exception as e:
                    log_exc_traceback_as_error()
                    self.configure_socket.send_json({
                        "success": False,
                        "error": unicode(e)
                    })

            elif sock == self.shutdown_socket:
                # First message part is just the topic name
                sock.recv()
                msg = sock.recv_json()
                self.monitors.shutdown_and_remove(msg['monitor_id'])

    def send_datapoint(self, dp):
        """
        @param dp - Python dict in the form of golib's datapoint.Datapoint
        """
        with self.datapoints_lock:
            self.datapoints_socket.send(DATAPOINTS_TOPIC, zmq.SNDMORE)
            self.datapoints_socket.send_json(dp.to_message_dict())


class ZMQHandler(logging.Handler):
    def __init__(self, socket, topic='logs'):
        self.socket = socket
        self.topic = topic

        super(ZMQHandler, self).__init__()

    def emit(self, record):
        self.socket.send(self.topic, zmq.SNDMORE)
        self.socket.send_json({
            "message": record.message,
            "logger": record.name,
            "source_path": record.pathname,
            "lineno": record.lineno,
            "created": record.created,
            "level": record.levelname,
        })

def log_exc_traceback_as_error():
    exc_type, exc_value, exc_traceback = sys.exc_info()
    logging.error(traceback.format_exception(exc_type, exc_value, exc_traceback))
