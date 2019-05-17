#!/usr/bin/env python3
import sys
from ruamel import yaml

# Generated this from `jq -r '[.[] | .metric_name]' < hostbased_filters.json`
# It's just a global list of metrics whitelisted by any monitor. Maybe use
# the MTS categorizer somehow.
with open("metrics.json") as f:
    whitelist = set(yaml.safe_load(f))

metaFile = sys.argv[1]

with open(metaFile) as f:
    meta = yaml.round_trip_load(f)

for mon in meta["monitors"]:
    for metric, info in mon["metrics"].items():
        info["included"] = metric in whitelist

with open(metaFile, "wt") as f:
    yaml.round_trip_dump(meta, f)