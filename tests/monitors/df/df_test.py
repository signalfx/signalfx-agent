import pytest

from tests.helpers import verify
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_log_message
from tests.helpers.metadata import Metadata
from tests.helpers.util import wait_for
from tests.helpers.verify import verify_included_metrics

pytestmark = [pytest.mark.collectd, pytest.mark.df, pytest.mark.monitor_without_endpoints]

METADATA = Metadata.from_package("collectd/df")


# def test_df_included():
#     verify_included_metrics(
#         """
#     enableExtraMetricsFilter: true
#     monitors:
#       - type: collectd/df
#         hostFSPath: /
#     """
#     )
#
#
# def test_df_report_inodes():
#     verify(
#         """
#         enableExtraMetricsFilter: true
#         monitors:
#           - type: collectd/df
#             hostFSPath: /
#             reportInodes: true
#         """,
#         METADATA.included_metrics | {"df_inodes.x", "df_inodes.y"},
#     )
#
#
# def test_df_report_percentage():
#     expected_metrics = METADATA.all_metrics
#     with Agent.run(
#         """
#         enableExtraMetricsFilter: true
#         monitors:
#           - type: collectd/df
#             hostFSPath: /
#             valuesPercentage: true
#         """
#     ) as agent:
#         _ = (
#             wait_for(lambda: set(agent.fake_services.datapoints_by_metric) == expected_metrics),
#             "timed out waiting for metrics and/or dimensions!",
#         )
#         assert set(agent.fake_services.datapoints_by_metric) == expected_metrics
#         assert not has_log_message(agent.output.lower(), "error"), "error found in agent output!"
#
#
# def test_df_report_all():
#     """
#     enableExtraMetricsFilter: true
#     monitors:
#       - type: collectd/df
#         hostFSPath: /
#         valuesPercentage: true
#         reportInodes: true
#     """
#     METADATA.all_metrics
