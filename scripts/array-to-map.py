#!/usr/bin/env python3

# Temporary script to migrate metrics to key based names.
import glob
from pathlib import Path

from ruamel import yaml


def main():
    for metadata in glob.glob("pkg/monitors/**/metadata.yaml", recursive=True):
        path = Path(metadata)
        meta = yaml.round_trip_load(path.read_text())
        for monitor in meta["monitors"]:
            # Skips monitor if stuff has been merged in to prevent duplicating.
            if monitor.merge:
                continue

            try:
                metrics_map = {
                    m["name"]: {"description": m.get("description"), "included": m.get("included"), "type": m["type"]}
                    for m in monitor.get("metrics") or []
                }
                dim_map = {d["name"]: {"description": d["description"]} for d in monitor.get("dimensions") or []}
                prop_map = {
                    p["name"]: {"description": p["description"], "dimension": p["dimension"]}
                    for p in monitor.get("properties") or []
                }
                group_map = {g["name"]: {"description": g["description"]} for g in monitor.get("groups") or []}

            except:
                print(f"""failed to process {monitor["monitorType"]} in {path}""")
                raise

            if monitor.get("metrics"):
                monitor["metrics"] = metrics_map
            if monitor.get("dimensions"):
                monitor["dimensions"] = dim_map
            if monitor.get("properties"):
                monitor["properties"] = prop_map
            if monitor.get("groups"):
                monitor["groups"] = group_map

        with open(path, "w") as f:
            yaml.round_trip_dump(meta, f)


if __name__ == "__main__":
    main()
