from collections import defaultdict

import yaml
from tests.paths import REPO_ROOT_DIR


class Metadata:
    """Metadata information like metrics and dimensions for a monitor"""

    def __init__(self, monitor_type, default_metrics, nondefault_metrics, metrics_by_group, dims):
        self.monitor_type = monitor_type
        self.default_metrics = default_metrics
        self.nondefault_metrics = nondefault_metrics
        self.all_metrics = self.default_metrics | self.nondefault_metrics
        self.metrics_by_group = metrics_by_group
        self.dims = dims

    @classmethod
    def from_package(cls, monitor_package_path, mon_type=None):
        with open(
            REPO_ROOT_DIR / "internal" / "monitors" / monitor_package_path / "metadata.yaml", "r", encoding="utf-8"
        ) as fd:
            doc = yaml.safe_load(fd)
            monitor = _find_monitor(doc["monitors"], mon_type)

            metrics_by_group = defaultdict(set)
            for metric, info in (monitor.get("metrics") or {}).items():
                group = info.get("group")
                if group:
                    metrics_by_group[group].add(metric)

            return cls(
                monitor_type=monitor["monitorType"],
                default_metrics=frozenset(_get_monitor_metrics(monitor, default=True)),
                nondefault_metrics=frozenset(_get_monitor_metrics(monitor, default=False)),
                metrics_by_group=metrics_by_group,
                dims=frozenset(_get_monitor_dims(doc, mon_type)),
            )


def _find_monitor(monitors, mon_type):
    if len(monitors) == 1:
        return monitors[0]

    if mon_type is None:
        raise ValueError("mon_type kwarg must be provided when there is more than one monitor in a metadata.yaml file")

    for monitor in monitors:
        if monitor["monitorType"] == mon_type:
            return monitor

    raise ValueError(f"mon_type {mon_type} was not found")


def _get_monitor_metrics(monitor, default):
    for metric, info in (monitor.get("metrics") or {}).items():
        if info["default"] == default:
            yield metric


def _get_monitor_dims(doc, mon_type=None):
    def filter_dims(dimensions):
        return (dimensions or {}).keys()

    monitors = doc["monitors"]

    if len(monitors) == 1:
        return filter_dims(monitors[0].get("dimensions"))

    if mon_type is None:
        raise ValueError("mon_type kwarg must be provided when there is more than one monitor in a metadata.yaml file")

    for monitor in monitors:
        if monitor["monitorType"] == mon_type:
            return filter_dims(monitor.get("dimensions"))

    raise ValueError(f"mon_type {mon_type} was not found")
