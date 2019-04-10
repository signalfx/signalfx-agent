import randommodule
from sfxrunner.scheduler.simple import SimpleScheduler


def run(config, output):
    output.send_gauge("my.gauge", 1, {"a": config["a"]})
