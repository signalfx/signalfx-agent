import asyncio
import json
import random
import string
import time
from base64 import b64encode
from contextlib import contextmanager
from functools import partial as p

from sanic import Sanic, response
from signalfx.generated_protocol_buffers import signal_fx_protocol_buffers_pb2 as sf_pbuf
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint, has_no_datapoint, has_time_series
from tests.helpers.util import ensure_always, run_simple_sanic_app, wait_for


@contextmanager
def run_fake_uaa():
    app = Sanic(name="".join(random.choices(string.ascii_lowercase, k=16)))

    token = "good-token"

    def set_token(new_token):
        nonlocal token
        token = new_token

    @app.post("/oauth/token")
    async def get_token(req):  # pylint:disable=unused-variable
        auth_value = req.headers.get("Authorization")
        expected_auth = b"Basic " + b64encode(b"myusername:mypassword")
        if expected_auth == auth_value.encode("utf-8"):
            json_data = {
                "access_token": token,
                "token_type": "bearer",
                "expires_in": 1_000_000,
                "scope": "",
                "jti": "28edda5c-4e37-4a63-9ba3-b32f48530a51",
            }
            return response.json(json_data)
        return response.text("Unauthorized", status=401)

    with run_simple_sanic_app(app) as url:
        yield url, set_token


@contextmanager
def run_fake_rlp_gateway(envelopes):
    app = Sanic(name="".join(random.choices(string.ascii_lowercase, k=16)))

    expected_token = "good-token"

    def set_token(token):
        nonlocal expected_token
        expected_token = token

    @app.route("/v2/read", stream=True)
    async def stream_envelopes(req):  # pylint:disable=unused-variable
        auth_value = req.headers.get("Authorization")

        def is_auth_valid():
            expected_auth = b"bearer " + expected_token.encode("utf-8")
            return expected_auth == auth_value.encode("utf-8")

        if not is_auth_valid():
            return response.text("Unauthorized (bad token)", status=401)

        async def streaming(resp):
            while True:
                if not is_auth_valid():
                    return

                for e in envelopes:
                    data = b"data: " + json.dumps(e).encode("utf-8") + b"\n\n"
                    await resp.write(data)
                await asyncio.sleep(1)

        return response.stream(streaming)

    with run_simple_sanic_app(app) as url:
        yield url, set_token


def test_pcf_nozzle():
    firehose_envelopes = [
        {
            "batch": [
                {
                    "timestamp": "1580228407476075606",
                    "source_id": "uaa",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"vitals.jvm.cpu.load": {"unit": "gauge", "value": 0}}},
                }
            ]
        },
        {
            "batch": [
                {
                    "timestamp": "1580228407476126130",
                    "source_id": "uaa",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"vitals.jvm.thread.count": {"unit": "gauge", "value": 47}}},
                }
            ]
        },
        {
            "batch": [
                {
                    "timestamp": "1580228407476264719",
                    "source_id": "uaa",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"vitals.jvm.non-heap.init": {"unit": "gauge", "value": 7_667_712}}},
                }
            ]
        },
        {
            "batch": [
                {
                    "timestamp": "1580428783743352757",
                    "source_id": "doppler",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "direction": "egress",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "counter": {"name": "dropped", "delta": "0", "total": "149000"},
                },
                {
                    "timestamp": "1580428783743352757",
                    "source_id": "712c7c06-62eb-4cd4-92e3-1a58683d1866",
                    "instance_id": "0",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "compute",
                        "origin": "rep",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"cpu": {"unit": "gauge", "value": 555}}},
                },
            ]
        },
        {
            "batch": [
                {
                    "timestamp": "1580428783743496839",
                    "source_id": "doppler",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "direction": "ingress",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "counter": {"name": "dropped", "delta": "0", "total": "0"},
                },
                {
                    "timestamp": "1580428783744624100",
                    "source_id": "doppler",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "counter": {"name": "egress", "delta": "0", "total": "1075978016"},
                },
                {
                    "timestamp": "1580428783744924877",
                    "source_id": "doppler",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "counter": {"name": "egress", "delta": "7457", "total": "1075985473"},
                },
                {
                    "timestamp": "1580428783833603896",
                    "source_id": "system_metrics_agent",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "service-instance_a474d20d-9a64-4bac-993d-e0f644604083",
                        "index": "2cc60900-9c39-4ec3-80bb-2c1d22c10130",
                        "ip": "10.0.8.28",
                        "job": "mongodb-config-agent",
                        "origin": "system_metrics_agent",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"system_cpu_sys": {"unit": "Percent", "value": 0.315_324_026_576_374_3}}},
                },
                {
                    "timestamp": "1580428783833625031",
                    "source_id": "system_metrics_agent",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "service-instance_a474d20d-9a64-4bac-993d-e0f644604083",
                        "index": "2cc60900-9c39-4ec3-80bb-2c1d22c10130",
                        "ip": "10.0.8.28",
                        "job": "mongodb-config-agent",
                        "origin": "system_metrics_agent",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {
                        "metrics": {
                            "system_disk_ephemeral_inode_percent": {"unit": "Percent", "value": 0.103_100_393_700_787_4}
                        }
                    },
                },
            ]
        },
    ]
    with run_fake_rlp_gateway(firehose_envelopes) as [gateway_url, _], run_fake_uaa() as [uaa_url, _]:
        with Agent.run(
            f"""
        disableHostDimensions: true
        monitors:
         - type: cloudfoundry-firehose-nozzle
           uaaUrl: {uaa_url}
           rlpGatewayUrl: {gateway_url}
           uaaUser: myusername
           uaaPassword: mypassword
           extraMetrics:
            - "*"
                """
        ) as agent:
            expected_time_series = [
                [
                    "uaa.vitals.jvm.non-heap.init",
                    {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "source_id": "uaa",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "uaa.vitals.jvm.cpu.load",
                    {
                        "source_id": "uaa",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "uaa.vitals.jvm.thread.count",
                    {
                        "source_id": "uaa",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "doppler.dropped",
                    {
                        "source_id": "doppler",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "direction": "egress",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "doppler.dropped",
                    {
                        "source_id": "doppler",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "direction": "ingress",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "rep.cpu",
                    {
                        "source_id": "712c7c06-62eb-4cd4-92e3-1a58683d1866",
                        "instance_id": "0",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "compute",
                        "origin": "rep",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "doppler.egress",
                    {
                        "source_id": "doppler",
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "metric_version": "2.0",
                        "origin": "loggregator.doppler",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "system_metrics_agent.system_cpu_sys",
                    {
                        "source_id": "system_metrics_agent",
                        "deployment": "service-instance_a474d20d-9a64-4bac-993d-e0f644604083",
                        "index": "2cc60900-9c39-4ec3-80bb-2c1d22c10130",
                        "ip": "10.0.8.28",
                        "job": "mongodb-config-agent",
                        "origin": "system_metrics_agent",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
                [
                    "system_metrics_agent.system_disk_ephemeral_inode_percent",
                    {
                        "source_id": "system_metrics_agent",
                        "deployment": "service-instance_a474d20d-9a64-4bac-993d-e0f644604083",
                        "index": "2cc60900-9c39-4ec3-80bb-2c1d22c10130",
                        "ip": "10.0.8.28",
                        "job": "mongodb-config-agent",
                        "origin": "system_metrics_agent",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                ],
            ]

            for metric_name, dimensions in expected_time_series:
                assert wait_for(p(has_time_series, agent.fake_services, metric_name=metric_name, dimensions=dimensions))

            assert has_datapoint(agent.fake_services, metric_type=sf_pbuf.GAUGE, metric_name="uaa.vitals.jvm.cpu.load")
            assert has_datapoint(
                agent.fake_services, metric_type=sf_pbuf.CUMULATIVE_COUNTER, metric_name="doppler.dropped"
            )


def test_restarts_on_token_expiration():
    firehose_envelopes = [
        {
            "batch": [
                {
                    "timestamp": "1580228407476075606",
                    "source_id": "uaa",
                    "instance_id": "",
                    "deprecated_tags": {},
                    "tags": {
                        "deployment": "cf-389cbac3d7a2c6c990c8",
                        "index": "ba5499ed-129c-48f2-877c-e270e5bd2648",
                        "ip": "10.0.4.7",
                        "job": "control",
                        "origin": "uaa",
                        "product": "Small Footprint Pivotal Application Service",
                        "system_domain": "sys.industry.cf-app.com",
                    },
                    "gauge": {"metrics": {"vitals.jvm.cpu.load": {"unit": "gauge", "value": 0}}},
                }
            ]
        }
    ]

    expected_metric = "uaa.vitals.jvm.cpu.load"

    with run_fake_rlp_gateway(firehose_envelopes) as [gateway_url, set_gateway_token], run_fake_uaa() as [
        uaa_url,
        set_uaa_token,
    ]:
        with Agent.run(
            f"""
        disableHostDimensions: true
        monitors:
         - type: cloudfoundry-firehose-nozzle
           uaaUrl: {uaa_url}
           rlpGatewayUrl: {gateway_url}
           uaaUser: myusername
           uaaPassword: mypassword
           extraMetrics:
            - "*"
                """
        ) as agent:
            assert wait_for(p(has_datapoint, agent.fake_services, metric_name=expected_metric))

            set_gateway_token("different-token")
            time.sleep(2)
            agent.fake_services.reset_datapoints()

            assert ensure_always(
                p(has_no_datapoint, agent.fake_services, metric_name=expected_metric), timeout_seconds=10
            )

            set_uaa_token("different-token")

            assert wait_for(p(has_datapoint, agent.fake_services, metric_name=expected_metric))
