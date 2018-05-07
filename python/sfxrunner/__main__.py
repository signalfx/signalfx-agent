from __future__ import absolute_import

import bdb
import logging
import logging.config
import os
import sys

from .logging import PipeLogHandler, log_exc_traceback_as_error
from .runner import Runner

logging.config.dictConfig({
    "version": 1,
    "formatters": {},
    "filters": {},
    "handlers": {},
    "loggers": {},
})

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

runner = Runner(input_reader, output_writer)

logger.info("Starting up Python monitor runner")

try:
    runner.run()
except (KeyboardInterrupt, SystemExit, bdb.BdbQuit):
    sys.exit(1)
except Exception as e:  # pylint: disable=broad-except
    log_exc_traceback_as_error()
