import os
import pytest

from tests.kubernetes.utils import (
    run_k8s_monitors_test,
    get_metrics_from_doc,
    get_dims_from_doc,
)

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]


@pytest.mark.k8s
@pytest.mark.kubernetes
def test_df_in_k8s(agent_image, minikube, k8s_test_timeout, k8s_namespace):
    monitors = [
        {"type": "collectd/df",
         "hostFSPath": "/hostfs"},
    ]
    run_k8s_monitors_test(
        agent_image,
        minikube,
        monitors,
        namespace=k8s_namespace,
        expected_metrics=get_metrics_from_doc("collectd-df.md"),
        expected_dims=get_dims_from_doc("collectd-df.md"),
        test_timeout=k8s_test_timeout)

