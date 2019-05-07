#!/usr/bin/env python3
import argparse
import sys
import time
from pathlib import Path

from tests.helpers.agent import Agent

sys.path.insert(0, str(Path(__file__).parent.parent.resolve()))


def main(opts):

    config = Path(opts.config).read_text()

    with Agent.run(config, debug=False) as agent:
        time.sleep(opts.period)
        if opts.enable_metrics:
            print("Metrics:")
            print("\n".join(sorted(set(agent.fake_services.datapoints_by_metric))))
        if opts.enable_dimensions:
            if opts.enable_metrics:
                print()
            print("Dimensions:")
            print("\n".join(sorted(set(agent.fake_services.datapoints_by_dim))))


if __name__ == "__main__":
    args = argparse.ArgumentParser(
        formatter_class=argparse.RawDescriptionHelpFormatter,
        description="""
Runs agent and prints out metrics and dimensions emitted. Config file example:

    monitors:
      - type: collectd/df
        hostFSPath: /
""",
    )
    args.add_argument("--config", "-c", type=str, metavar="PATH", required=True, help="Agent monitors configuration")
    args.add_argument(
        "-p", "--period", type=int, metavar="SECONDS", default=10, help="Collection period (default: %(default)s)"
    )
    args.add_argument(
        "--without-metrics",
        "-wm",
        action="store_false",
        default=True,
        dest="enable_metrics",
        help="Disable metric output",
    )
    args.add_argument(
        "--without-dimensions",
        "-wd",
        action="store_false",
        default=True,
        dest="enable_dimensions",
        help="Disable dimension output",
    )
    opts = args.parse_args()
    main(opts)
