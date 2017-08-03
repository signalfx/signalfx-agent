# Adapts Datadog Python plugins to the neo-agent.

# DD "checks" are fairly similar to our monitors, except that their
# configuration is split up across three variables, `init_config`,
# `agentConfig`, and `instances`.  `init_config` and `agentConfig` both are
# covered by our monitor config, and the `instances` config roughly corresponds
# to a "service" in our agent.

# There does not appear to be the equivalent of a "static" check that
# corresponds to our static monitors.  For checks that are intended to have
# only one set of configuration for (e.g. the btrfs check), they do it by
# enforcing that only one instance can be specified.

# `init_config` appears to be a default config that applies to all instances,
# to which the check will fall back to if the specific instance does not
# specify a particular config option.

import imp
import inspect
import os
import sys

from .datapoint import Datapoint, GAUGE, CUMULATIVE_COUNTER, COUNTER
from .wrapper import MonitorWrapper
from .rfc3339 import timestamptostr

dd_core_integrations_path = "/opt/dd/integrations-core"
dd_agent_dirpath = "/opt/dd/dd-agent"

sys.path.append(dd_agent_dirpath)

MONITOR_TYPE_PREFIX = 'dd/'

DISABLED_CHECKS = [
    "wmi_check",
    "windows_service",
    "iis",
    "win32_event_log",
]

dd_type_to_signalfx_type = {
    'gauge': GAUGE,
    'counter': CUMULATIVE_COUNTER,
    'count': COUNTER,
}


def load_check_class(dirpath):
    from checks import AgentCheck
    module = imp.load_source(os.path.basename(dirpath), os.path.join(dirpath, 'check.py'))
    classes = inspect.getmembers(module, inspect.isclass)

    for _, cls in classes:
        if issubclass(cls, AgentCheck) and AgentCheck in cls.__bases__:
            return cls
    return None

def get_check_dirs():
    d = dd_core_integrations_path
    for o in os.listdir(d):
        dirpath = os.path.join(d,o)

        is_dir = os.path.isdir(dirpath)
        has_check_file = os.path.isfile(os.path.join(dirpath, 'check.py'))
        is_excluded = any([dc in dirpath for dc in DISABLED_CHECKS])

        if is_dir and has_check_file and not is_excluded:
            yield os.path.join(d,o)


class DataDogCheckFactory(object):
    def __init__(self):
        self.checks = dict()

    def _load_all_check_classes(self):
        for d in get_check_dirs():
            cls = load_check_class(d)
            if cls is not None:
                self.checks[os.path.basename(d)] = cls

    def get_all_check_names(self):
        if not len(self.checks):
            self._load_all_check_classes()

        return self.checks.keys()

    def create(self, name, config):
        try:
            check_cls = self.checks[name]
        except KeyError:
            logger.error("No such Datadog check: %s" % name)
            return None

        return check_cls(name, config.get('init_config', {}), config.get('agentConfig', {}), config.get('instances', []))

check_factory = DataDogCheckFactory()

class DataDogMonitorWrapper(MonitorWrapper):
    def __init__(self, config, send_datapoint):
        super(DataDogMonitorWrapper, self).__init__(config, send_datapoint)

        # We assume all datadog monitor types start with "dd/..."
        check_name = config['Type'].split('/')[1]

        self.instance = check_factory.create(check_name, config)

    def get_datapoints(self):
        self.instance.run()
        metrics = self.instance.get_metrics()
        for m in metrics:
            yield self.convert_metric_to_datapoint(m)

    def convert_metric_to_datapoint(self, metric):
        metric_name, timestamp, value, tags = metric
        metric_type = tags.pop('type')

        nested_tags = tags.pop('tags', [])
        dimensions = dict(tags)
        for t in nested_tags:
            k, v = t.split(":")
            dimensions[k] = v

        return Datapoint(
            monitor_id=self.config['Id'],
            metric=metric_name,
            metric_type=dd_type_to_signalfx_type.get(metric_type, GAUGE),
            value=value,
            timestamp=timestamptostr(timestamp),
            dimensions=dimensions)

