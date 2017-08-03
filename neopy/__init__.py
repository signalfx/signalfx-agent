import json
import logging
import sys
import traceback
import zmq

from .monitors import Monitors

DATAPOINTS_TOPIC = "datapoints"

class NeoPy(object):
    def __init__(self, register_path, configure_path, shutdown_path, datapoints_path):
        self.zmq_ctx = zmq.Context.instance()

        self.register_socket = self.zmq_ctx.socket(zmq.REP)
        self.register_path = register_path

        self.configure_socket = self.zmq_ctx.socket(zmq.REP)
        self.configure_path = configure_path

        self.shutdown_socket = self.zmq_ctx.socket(zmq.SUB)
        self.shutdown_path = shutdown_path

        self.datapoints_socket = self.zmq_ctx.socket(zmq.PUB)
        self.datapoints_path = datapoints_path

    def run(self):
        self.register_socket.connect(self.register_path)

        self.configure_socket.connect(self.configure_path)

        self.shutdown_socket.connect(self.shutdown_path)
        self.shutdown_socket.subscribe("shutdown")

        self.datapoints_socket.connect(self.datapoints_path)

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

    def process_messages(self, rlist):
        for sock in rlist:
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
                # First message is just the topic name
                sock.recv()
                msg = sock.recv_json()
                self.monitors.shutdown_and_remove(msg['monitor_id'])

    # @param dp - datapoint.Datapoint instance
    def send_datapoint(self, dp):
        self.datapoints_socket.send(DATAPOINTS_TOPIC, zmq.SNDMORE)
        self.datapoints_socket.send_json(dp.to_message_dict())


def log_exc_traceback_as_error():
    exc_type, exc_value, exc_traceback = sys.exc_info()
    logging.error(traceback.format_exception(exc_type, exc_value, exc_traceback))
