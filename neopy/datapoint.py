GAUGE = 0
COUNTER = 1
ENUM = 2
CUMULATIVE_COUNTER = 3
RATE = 4
TIMESTAMP = 5

class Datapoint(object):
    def __init__(self, monitor_id, metric, metric_type, timestamp, value,
                 dimensions):
        self.monitor_id = monitor_id
        self.metric = metric
        self.metric_type = metric_type
        self.timestamp = timestamp
        self.value = value
        self.dimensions = dimensions

    def to_message_dict(self):
        return {
            "monitor_id": self.monitor_id,
            "datapoint": {
                "metric": self.metric,
                "metric_type": self.metric_type,
                "timestamp": self.timestamp,
                "value": self.value,
                "dimensions": self.dimensions,
            }
        }
