"""
Test that the cluster property is synced to host id dimensions
"""
from tests.helpers.agent import Agent, ensure_fake_backend
from tests.helpers.util import wait_for_assertion


def test_cluster_prop_is_merged_into_existing():
    with ensure_fake_backend() as fake_services:
        fake_services.dims["host"]["myhost"] = {"customProperties": {"os": "linux"}, "tags": ["important"]}

        with Agent.run(
            """
            cluster: prod
            hostname: myhost
            writer:
              propertiesSendDelaySeconds: 1
        """,
            fake_services=fake_services,
            # Make it ignore all platform metrics for CI environments
            extra_env={"SKIP_PLATFORM_HOST_DIMS": "yes"},
        ):

            def assert_cluster_property():
                dim = fake_services.dims["host"]["myhost"]
                assert dim["customProperties"] == {"os": "linux", "cluster": "prod"}
                assert dim["tags"] == ["important"]

            wait_for_assertion(assert_cluster_property)


def test_cluster_prop_is_added_to_host_dims():
    with Agent.run(
        """
        cluster: prod
        hostname: myhost
        writer:
          propertiesSendDelaySeconds: 1
    """,
        # Make it ignore all platform metrics for CI environments
        extra_env={"SKIP_PLATFORM_HOST_DIMS": "yes"},
    ) as agent:

        def assert_cluster_property():
            assert "myhost" in agent.fake_services.dims["host"]
            dim = agent.fake_services.dims["host"]["myhost"]
            assert dim["customProperties"] == {"cluster": "prod"}
            assert dim["tags"] in [None, []]

        wait_for_assertion(assert_cluster_property)


def test_cluster_prop_sync_cluster_on_host_dim():
    with Agent.run(
        """
        cluster: prod
        hostname: myhost
        syncClusterOnHostDimension: true
        writer:
          propertiesSendDelaySeconds: 1
    """
    ) as agent:

        def assert_cluster_property():
            assert "myhost" in agent.fake_services.dims["host"]
            dim = agent.fake_services.dims["host"]["myhost"]
            assert dim["customProperties"] == {"cluster": "prod"}
            assert dim["tags"] in [None, []]

        wait_for_assertion(assert_cluster_property)
