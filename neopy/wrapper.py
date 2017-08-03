import logging
from threading import Timer

DEFAULT_INTERVAL_SECONDS = 10

class MonitorWrapper(object):
    def __init__(self, config, send_datapoint):
        self.config = config
        self.send_datapoint = send_datapoint

    def start_getting_metrics(self):
        logging.info("Pulling metrics for %s" % self.config['Type'])
        self._get_and_send_metrics()

        interval = self.config.get('intervalSeconds', DEFAULT_INTERVAL_SECONDS)
        self.timer = Timer(interval, self.start_getting_metrics)
        self.timer.daemon = True
        self.timer.start()

    def _get_and_send_metrics(self):
        for dp in self.get_datapoints():
            print "sending dp"
            self.send_datapoint(dp)

    def get_datapoints(self):
        raise NotImplementedError("Should be implemented by monitor wrapper implementations")

    def shutdown(self):
        if self.timer:
            self.timer.cancel()
