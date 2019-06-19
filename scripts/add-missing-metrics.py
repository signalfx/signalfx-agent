#!/usr/bin/env python3
#
# Super hacky script to add missing metrics from a json dump. After running this consider
# running sync-included-status.py to update the included field.
#
import sys
from ruamel import yaml

# Use format from dump_json() here.
json_dump = {
    "metrics": {"counter.hadoop.cluster.metrics.total_mb": {"type": "CUMULATIVE_COUNTER"}},
    "dimensions": ["dsname:value"],
}


def map_metric_type(typ):
    return dict(CUMULATIVE_COUNTER="cumulative", GAUGE="gauge")[typ]


metaFile = sys.argv[1]

with open(metaFile) as f:
    meta = yaml.round_trip_load(f)

assert len(meta["monitors"]) == 1, "only supports 1 monitor"
monitor = meta["monitors"][0]
metrics = monitor["metrics"]

for metric, info in json_dump["metrics"].items():
    typ = map_metric_type(info["type"])

    metrics.setdefault(metric, {"description": None, "type": typ, "included": False})
    assert metrics[metric]["type"] == typ, f"{metric} type doesn't match observed metric type {typ}"

with open(metaFile, "wt") as f:
    yaml.round_trip_dump(meta, f)
