import logging

from . import datadog
from .scheduler import IntervalScheduler


class Monitors(object):
    def __init__(self, send_datapoint):
        self.monitors_by_id = dict()
        self.scheduler = IntervalScheduler()
        self.send_datapoint = send_datapoint

    @property
    def registered_monitors(self):
        monitor_names = []

        monitor_names.extend([
            "dd/%s" % n for n in datadog.check_factory.get_all_check_names()])

        return monitor_names

    # When we get a request to reconfigure a monitor, we just destroy it and
    # recreate it.
    def configure(self, config):
        logging.info("Configuring monitor: %s" % config)
        monitor_id = config['Id']

        if monitor_id in self.monitors_by_id:
            self.shutdown_and_remove(monitor_id)

        instance = self.create_instance(config)
        self.monitors_by_id[monitor_id] = instance

        instance.start_getting_metrics()
        return True

    def create_instance(self, config):
        if config['Type'].startswith(datadog.MONITOR_TYPE_PREFIX):
            return datadog.DataDogMonitorWrapper(config, self.scheduler, self.send_datapoint)

    def shutdown_and_remove(self, monitor_id):
        if monitor_id not in self.monitors_by_id:
            return

        mon = self.monitors_by_id[monitor_id]
        mon.shutdown()

        del self.monitors_by_id[monitor_id]

        logging.info("Shut down monitor: %s" % monitor_id)

