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

    def get_with_matching_dim(self, dim_key, dim_value):
        """
        Returns a dict mapping from an internal metric name to its value with dimensions.
        """
        resp = requests.get(f"http://{self.host}:{self.port}/metrics")
        metrics = resp.json()
        ret = {}
        for met in metrics:
            dim = met.get("dimensions", {}).get(dim_key, None)
            if dim == dim_value:
                ret[met["metric"]] = met["value"]
        return ret
