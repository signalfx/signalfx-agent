#!/usr/bin/env python

# Temporary script to migrate metrics and monitor docs to yaml.

import yaml
import json

from collections import defaultdict

WHITELIST = {}

def str_presenter(dumper, data):
  if '\n' in data:
    return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='|')
  return dumper.represent_scalar('tag:yaml.org,2002:str', data)

yaml.add_representer(str, str_presenter)
yaml.add_representer(unicode, str_presenter)

with open("whitelist.json", "r") as f:
    data = json.load(f)
    for d in data:
        assert d["negated"]
        WHITELIST[d["monitorType"]] = set(d["metricNames"])

def getMetrics(monitorName, metrics):
    if metrics is None:
        return None

    whitelist = WHITELIST.get(monitorName, set())

    if not whitelist:
        print("WARNING: {} was not in the whitelist".format(monitorName))

    return [{
        "name": m["name"],
        "type": m["type"],
        "description": m["description"].strip(),
        "included": m["name"] in whitelist,
    } for m in metrics]

pkgs = defaultdict(lambda: [])

def main():
    with open("selfdescribe.json") as f:
        sd = json.load(f)

    for monitor in sd["Monitors"]:
        pkg = monitor["package"]
        out = {
            "monitorType": monitor["monitorType"],
            "doc": monitor["doc"].strip() + '\n',
            "metrics": getMetrics(monitor["monitorType"], monitor["metrics"]),
            "dimensions": monitor["dimensions"],
            "properties": monitor["properties"],
        }

        pkgs[pkg].append(out)

    for pkg, monitors in pkgs.iteritems():
        with open(pkg + "/metadata.yaml", 'w') as f:
            yaml.dump(monitors, f, encoding='utf-8', default_flow_style=False)

if __name__ == "__main__":
    main()