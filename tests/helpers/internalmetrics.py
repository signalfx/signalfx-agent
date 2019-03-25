import requests


class InternalMetricsClient:
    def __init__(self, host, port):
        self.host = host
        self.port = port

    def get(self):
        """
        Returns a dict mapping from an internal metric name to its value.
        """
        resp = requests.get(f"http://{self.host}:{self.port}/metrics")
        metrics = resp.json()
        return {m["metric"]: m["value"] for m in metrics}
