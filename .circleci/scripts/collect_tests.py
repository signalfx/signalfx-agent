import pytest
import sys

class CollectTests:
    def __init__(self):
        self.collected = set()

    def pytest_collection_finish(self, session):
        for item in session.items:
            if item.location[0] not in self.collected:
                print(item.location[0])
            self.collected.add(item.location[0])


markers = sys.argv[1]
directory = sys.argv[2]
pytest.main(['--collect-only', '-m', markers, '-p', 'no:terminal', directory], plugins=[CollectTests()])
