#!/usr/bin/env python3
import argparse
import time
from pathlib import Path

from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf

from tests.helpers.agent import Agent


def _get_metric_type(index):
    for metric_type, idx in sf_pbuf.MetricType.items():
        if int(index) == idx:
            return metric_type
    return str(index)


def main(opts):
    config = Path(opts.config).read_text()

    with Agent.run(config, debug=False) as agent:
        time.sleep(opts.period)
        if opts.enable_metrics:
            dps = [dp[0] for dp in agent.fake_services.datapoints_by_metric.values()]
            metrics = {(dp.metric, _get_metric_type(dp.metricType)) for dp in dps}

            print("Metrics:")
            for metric, metric_type in sorted(metrics):
                print(f"{metric_type:20} {metric}")
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
