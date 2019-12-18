import os

import pytest

if "K8S_VERSION" in os.environ and "K8S_MAX_VERSION" in os.environ:
    K8S_IS_LATEST = os.environ["K8S_VERSION"] == os.environ["K8S_MAX_VERSION"]
else:
    K8S_IS_LATEST = False

# Marker for only running kubernetes tests when it's the latest version.
LATEST = pytest.mark.skipif(not K8S_IS_LATEST, reason="Skipping because K8S_LATEST is false")
