from sfxrunner.scheduler.simple import SimpleScheduler


class Monitor(object):
    def __init__(self, output):
        self.output = output
        self.scheduler = SimpleScheduler()

    def configure(self, config):
        def gather():
            self.output.send_gauge("my.gauge", 1, {"a": config["a"]})

        self.scheduler.run_on_interval(config["intervalSeconds"], gather)
