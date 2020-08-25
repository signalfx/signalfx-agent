from functools import partial as p

import pytest
import yaml
from tests.helpers.assertions import has_all_dims, has_datapoint, has_dim_key, has_no_datapoint
from tests.helpers.kubernetes.utils import add_pod_spec_annotations
from tests.helpers.util import ensure_always, wait_for
from tests.paths import TEST_SERVICES_DIR


@pytest.mark.kubernetes
def test_k8s_api_observer_basic(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    with k8s_cluster.create_resources([nginx_yaml]):
        config = f"""
            observers:
             - type: k8s-api
            monitors:
             - type: collectd/nginx
               discoveryRule: 'discovered_by == "k8s-api" && kubernetes_namespace == "{k8s_cluster.test_namespace}" && port == 80 && container_spec_name == "nginx"'
               url: "http://{{{{.Host}}}}:{{{{.Port}}}}/nginx_status"
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx"}))


@pytest.mark.kubernetes
def test_config_from_annotations(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    nginx_with_annotations = yaml.dump_all(
        add_pod_spec_annotations(
            yaml.safe_load_all(nginx_yaml.read_bytes()),
            {
                "agent.signalfx.com/monitorType.http": "collectd/nginx",
                "agent.signalfx.com/config.80.extraDimensions": "{source: myapp}",
                "agent.signalfx.com/config.80.disableEndpointDimensions": "true",
                "agent.signalfx.com/config.80.datapointsToExclude": "[{metricName: nginx_connections.active}]",
            },
        )
    )
    with k8s_cluster.create_resources([nginx_with_annotations]):
        config = f"""
            observers:
             - type: k8s-api
            monitors: []
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx", "source": "myapp"}))
            assert ensure_always(
                p(
                    has_no_datapoint,
                    agent.fake_services,
                    metric_name="nginx_connections.active",
                    dimensions={"source": "myapp"},
                )
            )
            for dp in agent.fake_services.datapoints:
                if not has_all_dims(dp, {"source": "myapp"}):
                    continue
                assert not has_dim_key(dp, "kubernetes_pod_uid")


@pytest.mark.kubernetes
def test_merges_config_from_annotations_and_agent_yaml(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    nginx_with_annotations = yaml.dump_all(
        add_pod_spec_annotations(
            yaml.safe_load_all(nginx_yaml.read_bytes()),
            {
                "agent.signalfx.com/monitorType.http": "collectd/nginx",
                "agent.signalfx.com/config.80.extraDimensions": "{source: mytestapp}",
                "agent.signalfx.com/config.80.disableEndpointDimensions": "true",
            },
        )
    )
    with k8s_cluster.create_resources([nginx_with_annotations]):
        config = f"""
            observers:
             - type: k8s-api
            monitors:
             - type: internal-metrics
             - type: collectd/nginx
               discoveryRule: 'kubernetes_namespace == "{k8s_cluster.test_namespace}" && port == 80 && container_spec_name == "nginx"'
               datapointsToExclude:
                - metricName: nginx_requests
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx", "source": "mytestapp"})
            )
            for dp in agent.fake_services.datapoints:
                if not has_all_dims(dp, {"source": "mytestapp"}):
                    continue
                assert not has_dim_key(dp, "kubernetes_pod_uid")
            ensure_always(p(has_no_datapoint, agent.fake_services, metric_name="nginx_requests"), timeout_seconds=10)

            wait_for(p(has_datapoint, agent.fake_services, metric_name="sfxagent.active_monitors", value=2))


@pytest.mark.kubernetes
def test_k8s_annotations_in_discovery(k8s_cluster):
    nginx_yaml = TEST_SERVICES_DIR / "nginx/nginx-k8s.yaml"
    with k8s_cluster.create_resources([nginx_yaml]):
        config = """
            observers:
            - type: k8s-api

            monitors:
            - type: collectd/nginx
              discoveryRule: 'Get(kubernetes_annotations, "allowScraping") == "true" && port == 80'
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, dimensions={"plugin": "nginx"}))


@pytest.mark.kubernetes
def test_k8s_annotations_with_alt_ports(k8s_cluster):
    consul_yaml = list(yaml.safe_load_all((TEST_SERVICES_DIR / "consul/k8s.yaml").read_bytes()))
    # Remove the declared port so we make sure the additionalPortAnnotations is
    # what is causing it to be discovered.
    consul_yaml[0]["spec"]["template"]["spec"]["containers"][0]["ports"] = []

    with k8s_cluster.create_resources([yaml.dump_all(consul_yaml)]):
        config = """
            observers:
            - type: k8s-api
              additionalPortAnnotations:
               - prometheus.io/port

            monitors:
             - type: prometheus-exporter
               discoveryRule: 'Get(kubernetes_annotations, "prometheus.io/scrape") == "true" && Get(kubernetes_annotations, "prometheus.io/port") == ToString(port)'
               configEndpointMappings:
                 metricPath: 'Get(kubernetes_annotations, "prometheus.io/path", "/metrics")'
        """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="consul_runtime_malloc_count"))


@pytest.mark.kubernetes
def test_k8s_portless_pod(k8s_cluster):
    with k8s_cluster.create_resources([TEST_SERVICES_DIR / "redis" / "redis-k8s-portless.yaml"]):
        config = """
            observers:
            - type: k8s-api

            monitors:
            - type: collectd/redis
              discoveryRule: target == "pod" && kubernetes_pod_name =~ "redis-portless-only"
              port: 6379
         """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="bytes.used_memory_rss"))


@pytest.mark.kubernetes
def test_k8s_portless_pods_with_declared_port(k8s_cluster):
    with k8s_cluster.create_resources([TEST_SERVICES_DIR / "redis" / "redis-k8s.yaml"]):
        config = """
            observers:
            - type: k8s-api

            monitors:
            - type: collectd/redis
              discoveryRule: target == "pod" && kubernetes_pod_name =~ "redis-deployment"
              port: 6379
         """
        with k8s_cluster.run_agent(config) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name="bytes.used_memory_rss"))
