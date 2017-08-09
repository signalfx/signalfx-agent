import logging
from threading import Timer

DEFAULT_INTERVAL_SECONDS = 10

class MonitorWrapper(object):
    def __init__(self, config, scheduler, send_datapoint):
        self.config = config
        self.send_datapoint = send_datapoint
        self.scheduler = scheduler
        # Callback to stop scheduled datapoint gathering
        self.cancel = None

    def start_getting_metrics(self):
        interval = self.config.get('intervalSeconds', DEFAULT_INTERVAL_SECONDS)
        self.cancel = self.scheduler.run_on_interval(interval, self._get_and_send_metrics)

    def _get_and_send_metrics(self):
        logging.info("Pulling metrics for %s" % self.config['Type'])
        for dp in self.get_datapoints():
            print "sending dp"
            self.send_datapoint(dp)

    def get_datapoints(self):
        raise NotImplementedError("Should be implemented by monitor wrapper implementations")

    def shutdown(self):
        if self.cancel:
            self.cancel()
