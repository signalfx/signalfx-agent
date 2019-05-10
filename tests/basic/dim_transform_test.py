from functools import partial as p

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint
from tests.helpers.util import container_ip, run_service, wait_for


def test_endpoint_config_mapping():
    with run_service(
        "postgres", environment=["POSTGRES_USER=test_user", "POSTGRES_PASSWORD=test_pwd", "POSTGRES_DB=postgres"]
    ) as postgres_container:
        with Agent.run(
            f"""
          observers:
            - type: docker
          monitors:
            - type: postgresql
              host: {container_ip(postgres_container)}
              connectionString: "user=test_user password=test_pwd dbname=postgres sslmode=disable"
              port: 5432
              dimensionTransformations:
                database: db
          """
        ) as agent:
            assert wait_for(
                p(has_datapoint, agent.fake_services, dimensions={"db": "dvdrental"})
            ), "Didn't get properly transformed dimension name"

            assert not has_datapoint(agent.fake_services, dimensions={"database": "dvdrental"})
