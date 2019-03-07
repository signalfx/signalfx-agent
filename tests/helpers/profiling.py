import pathlib

import requests


class PProfClient:
    def __init__(self, host: str, port: int):
        self.host = host
        self.port = port

        pathlib.Path("/tmp/pprof").mkdir(parents=True, exist_ok=True)

    @property
    def _base_url(self):
        return f"http://{self.host}:{self.port}"

    def fetch_goroutines(self):
        resp = requests.get(f"{self._base_url}/debug/pprof/goroutine")
        return resp.content

    def save_goroutines(self):
        """
        Saves the pprof goroutine stack output to a tmpfile and returns the
        path
        """
        path = f"/tmp/pprof/goroutine.{self.host}-{self.port}"
        with open(path, "wb") as fd:
            print(f"Saving goroutines to {path}")
            fd.write(self.fetch_goroutines())

        return path
