import os
import sys
from pathlib import Path

REPO_ROOT_DIR = Path(__file__).parent.parent.resolve()
PROJECT_DIR = REPO_ROOT_DIR / "tests"
BUNDLE_DIR = Path(os.environ.get("BUNDLE_DIR", "/bundle"))
TEST_SERVICES_DIR = Path(os.environ.get("TEST_SERVICES_DIR", "/test-services"))
SELFDESCRIBE_JSON = REPO_ROOT_DIR / "selfdescribe.json"

if sys.platform == "win32":
    AGENT_BIN = Path(os.environ.get("AGENT_BIN", REPO_ROOT_DIR / "signalfx-agent.exe"))
else:
    AGENT_BIN = Path(os.environ.get("AGENT_BIN", "/bundle/bin/signalfx-agent"))
