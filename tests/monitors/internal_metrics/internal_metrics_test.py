from functools import partial as p
import os
import pytest
import string

from tests.helpers.util import wait_for, run_agent
from tests.helpers.assertions import *

from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.internal_metrics, pytest.mark.monitor_without_endpoints]


config = """
monitors:
  - type: internal-metrics

"""

def test_internal_metrics():
    with run_agent(config) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_metric_name, backend, "sfxagent.datapoints_sent")), "Didn't get internal metric datapoints"


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_internal_metrics_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "internal-metrics"}
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_metrics_from_doc("internal-metrics.md"),
        expected_dims=get_dims_from_doc("internal-metrics.md"),
        test_timeout=k8s_test_timeout)

