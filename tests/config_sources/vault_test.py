import json
import re
import time
from contextlib import contextmanager
from functools import partial as p
from textwrap import dedent

import hvac
from tests.helpers.assertions import has_datapoint, tcp_socket_open
from tests.helpers.util import container_ip, run_agent, run_container, wait_for

AUDIT_PREFIX = "AUDIT: "


@contextmanager
def run_vault():
    with run_container("vault:1.0.2") as vault_cont:
        vault_ip = container_ip(vault_cont)
        assert wait_for(p(tcp_socket_open, vault_ip, 8200), 30)
        assert wait_for(lambda: "Root Token:" in vault_cont.logs().decode("utf-8"), 10)

        logs = vault_cont.logs()
        token = re.search(r"Root Token: (.*)$", logs.decode("utf-8"), re.MULTILINE).group(1)
        assert token, "Could not get root token of vault server"
        client = hvac.Client(url=f"http://{vault_ip}:8200", token=token)
        client.sys.enable_audit_device(
            device_type="file", options={"log_raw": True, "prefix": AUDIT_PREFIX, "file_path": "stdout"}
        )
        yield [client, lambda: parse_audit_events_from_logs(vault_cont)]


def parse_audit_events_from_logs(vault_cont):
    return [
        json.loads(eline[len(AUDIT_PREFIX) :])
        for eline in vault_cont.logs().decode("utf-8").splitlines()
        if eline.startswith(AUDIT_PREFIX)
    ]


def audit_read_paths(audit_events):
    return [
        ae["request"].get("path")
        for ae in audit_events
        if ae["type"] == "request"
        and ae["request"].get("operation") == "read"
        and ae["request"]["path"] != "auth/token/lookup-self"
    ]


def audit_secret_renewals(audit_events):
    """
    Return the lease ids of all secret leases that have been renewed
    """
    return [
        ae["request"]["data"]["lease_id"]
        for ae in audit_events
        if ae["type"] == "request"
        and ae["request"].get("operation") == "update"
        and ae["request"].get("path") == "sys/leases/renew"
    ]


def audit_token_renewals(audit_events):
    """
    Return the accessors of the tokens renewals
    """
    return [
        ae["auth"]["accessor"]
        for ae in audit_events
        if ae["type"] == "request"
        and ae["request"].get("operation") == "update"
        and ae["request"].get("path") == "auth/token/renew-self"
    ]


def test_basic_vault_config():
    with run_vault() as [vault_client, get_audit_events]:
        vault_client.sys.enable_secrets_engine(backend_type="kv", options={"version": "1"})

        vault_client.write("secret/data/appinfo", data={"env": "prod"})
        vault_client.write("kv/usernames", app="me")
        with run_agent(
            dedent(
                f"""
            intervalSeconds: 2
            globalDimensions:
              env: {{"#from": "vault:secret/data/appinfo[data.env]"}}
              user: {{"#from": "vault:kv/usernames[app]"}}
            configSources:
              vault:
                vaultToken: {vault_client.token}
                vaultAddr: {vault_client.url}
            monitors:
             - type: collectd/uptime
        """
            )
        ) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"env": "prod"}))
            assert wait_for(p(has_datapoint, backend, dimensions={"user": "me"}))
            assert audit_read_paths(get_audit_events()) == ["secret/data/appinfo", "kv/usernames"], "expected two reads"


def test_vault_nonrenewable_secret_refresh():
    with run_vault() as [vault_client, get_audit_events]:
        vault_client.sys.enable_secrets_engine(backend_type="kv", options={"version": "1"})

        vault_client.write("kv/passwords", app="s3cr3t", ttl="10s")
        with run_agent(
            dedent(
                f"""
            intervalSeconds: 1
            globalDimensions:
              password: {{"#from": "vault:kv/passwords[app]"}}
            configSources:
              vault:
                vaultToken: {vault_client.token}
                vaultAddr: {vault_client.url}
            monitors:
             - type: internal-metrics
               metricsToExclude:
                - metricName: "!sfxagent.go_num_goroutine"
        """
            )
        ) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"password": "s3cr3t"}))
            assert audit_read_paths(get_audit_events()) == ["kv/passwords"], "expected one read"

            # Renew time is 1/2 of the lease time of 10s
            time.sleep(5)
            assert audit_read_paths(get_audit_events()) == ["kv/passwords", "kv/passwords"], "expected two reads"


def test_vault_renewable_secret_refresh():
    """
    Use the Mongo database secret engine to get renewable Mongo credentials to
    use in the Mongo collectd plugin.  Make sure the secret gets renewed as
    expected.
    """
    with run_container("mongo:3.6") as mongo_cont, run_vault() as [vault_client, get_audit_events]:
        assert wait_for(p(tcp_socket_open, container_ip(mongo_cont), 27017), 30), "mongo service didn't start"

        vault_client.sys.enable_secrets_engine(backend_type="database")

        vault_client.write(
            "database/config/my-mongodb-database",
            plugin_name="mongodb-database-plugin",
            allowed_roles="my-role",
            connection_url=f"mongodb://{container_ip(mongo_cont)}:27017/admin",
            username="admin",
            password="",
        )

        vault_client.write(
            "database/roles/my-role",
            db_name="my-mongodb-database",
            creation_statements='{ "db": "admin", "roles": [{ "role": "readWrite" }, {"role": "read", "db": "foo"}] }',
            default_ttl="13s",
            max_ttl="24h",
        )

        with run_agent(
            dedent(
                f"""
            intervalSeconds: 1
            configSources:
              vault:
                vaultToken: {vault_client.token}
                vaultAddr: {vault_client.url}
            monitors:
             - type: collectd/mongodb
               host: {container_ip(mongo_cont)}
               port: 27017
               databases:
                - admin
               username: {{"#from": "vault:database/creds/my-role[username]"}}
               password: {{"#from": "vault:database/creds/my-role[password]"}}
               metricsToExclude:
                - metricName: "!gauge.objects"
        """
            )
        ) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"plugin": "mongo"}))
            assert audit_read_paths(get_audit_events()) == ["database/creds/my-role"], "expected one read"

            time.sleep(10)
            assert audit_read_paths(get_audit_events()) == ["database/creds/my-role"], "expected still one read"

            renewals = audit_secret_renewals(get_audit_events())
            # The secret gets renewed immediately by the renewer and then again
            # within its lease duration period.
            assert len(renewals) == 2, "expected two renewal ops"
            for ren in renewals:
                assert "database/creds/my-role" in ren, "expected renewal of right secret"

            backend.datapoints.clear()
            assert wait_for(p(has_datapoint, backend, dimensions={"plugin": "mongo"})), "plugin lost access to mongo"


def test_vault_token_renewal():
    """
    Test the token renewal feature
    """
    with run_vault() as [vault_client, get_audit_events]:
        new_token = vault_client.create_token(policies=["root"], renewable=True, ttl="12s")

        vault_client.write("secret/data/appinfo", data={"env": "prod"})
        with run_agent(
            dedent(
                f"""
            intervalSeconds: 2
            globalDimensions:
              env: {{"#from": "vault:secret/data/appinfo[data.env]"}}
            configSources:
              vault:
                vaultToken: {new_token['auth']['client_token']}
                vaultAddr: {vault_client.url}
            monitors:
             - type: collectd/uptime
        """
            )
        ) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"env": "prod"}))
            assert audit_read_paths(get_audit_events()) == ["secret/data/appinfo"], "expected one reads"

            assert audit_token_renewals(get_audit_events()) == [
                new_token["auth"]["accessor"]
            ], "token immediately renews"

            time.sleep(10)

            assert audit_token_renewals(get_audit_events()) == [
                new_token["auth"]["accessor"],
                new_token["auth"]["accessor"],
            ], "token has renewed twice now"

            time.sleep(10)

            assert len(audit_token_renewals(get_audit_events())) >= 3, "token has renewed three times now"


def test_vault_kv_poll_refetch():
    """
    Test the KV v2 token refetch operation
    """
    with run_vault() as [vault_client, get_audit_events]:
        vault_client.write("secret/data/app", data={"env": "dev"})
        with run_agent(
            dedent(
                f"""
            intervalSeconds: 2
            globalDimensions:
               env: {{"#from": "vault:secret/data/app[data.env]"}}
            configSources:
              vault:
                vaultToken: {vault_client.token}
                vaultAddr: {vault_client.url}
                kvV2PollInterval: 10s
            monitors:
             - type: collectd/uptime
        """
            )
        ) as [backend, _, _]:
            assert wait_for(p(has_datapoint, backend, dimensions={"env": "dev"}))

            assert audit_read_paths(get_audit_events()) == ["secret/data/app"], "expected one read"

            vault_client.write("secret/data/app", data={"env": "prod"})
            assert wait_for(p(has_datapoint, backend, dimensions={"env": "prod"}))

            assert "secret/metadata/app" in audit_read_paths(get_audit_events())
