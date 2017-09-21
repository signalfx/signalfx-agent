import argparse
import logging
import sys

import neopy

logging.basicConfig(level=logging.INFO)

parser = argparse.ArgumentParser(
    description='NeoPy: Python <-> SignalFx NeoAgent')
parser.add_argument('--register-path', required=True)
parser.add_argument('--configure-path', required=True)
parser.add_argument('--shutdown-path', required=True)
parser.add_argument('--datapoints-path', required=True)
parser.add_argument('--logging-path', required=True)

args = parser.parse_args()

npy = neopy.NeoPy(
    register_path=args.register_path,
    configure_path=args.configure_path,
    shutdown_path=args.shutdown_path,
    datapoints_path=args.datapoints_path,
    logging_path=args.logging_path,
)

while True:
    try:
        npy.run()
    except (KeyboardInterrupt, SystemExit):
        sys.exit(1)
    except Exception as e:
        neopy.log_exc_traceback_as_error()
