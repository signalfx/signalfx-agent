from functools import partial as p
from textwrap import dedent
from io import BytesIO
import string

from requests import get, RequestException
import pytest

from tests.helpers.util import wait_for, run_agent, run_container, container_ip, get_docker_client
from tests.helpers.assertions import has_datapoint_with_dim


@pytest.fixture(scope='session')
def kong_image():
    dockerfile = BytesIO(bytes(dedent('''
        from kong:0.13-centos
        RUN yum install -y epel-release
        RUN yum install -y postgresql git
        WORKDIR /usr/local/share/lua/5.1/kong
        RUN sed -i '38ilua_shared_dict kong_signalfx_aggregation 10m;' templates/nginx_kong.lua
        RUN sed -i '38ilua_shared_dict kong_signalfx_locks 100k;' templates/nginx_kong.lua
        RUN sed -i '29i\ \ "signalfx",' constants.lua
        WORKDIR /opt/
        RUN git clone --depth 1 https://github.com/signalfx/kong-plugin-signalfx.git
        RUN cd kong-plugin-signalfx && luarocks make
        WORKDIR /
        RUN mkdir -p /usr/local/kong/logs
        RUN ln -s /dev/stderr /usr/local/kong/logs/error.log
        RUN ln -s /dev/stdout /usr/local/kong/logs/access.log
    '''), 'ascii'))
    client = get_docker_client()
    image, _ = client.images.build(fileobj=dockerfile, forcerm=True)
    try:
        yield image.short_id
    finally:
        client.images.remove(image=image.id, force=True)


def test_kong(kong_image):
    kong_env = dict(KONG_ADMIN_LISTEN='0.0.0.0:8001', KONG_LOG_LEVEL='warn', KONG_DATABASE='postgres',
                    KONG_PG_DATABASE='kong')

    with run_container('postgres:9.5', environment=dict(POSTGRES_USER='kong', POSTGRES_DB='kong')) as db:
        db_ip = container_ip(db)
        kong_env['KONG_PG_HOST'] = db_ip

        def db_is_ready():
            return db.exec_run('pg_isready -U postgres').exit_code == 0

        assert wait_for(db_is_ready)

        with run_container(kong_image, environment=kong_env, command='sleep inf') as migrations:

            def db_is_reachable():
                return migrations.exec_run('psql -h {} -U postgres'.format(db_ip)).exit_code == 0

            assert wait_for(db_is_reachable)
            assert migrations.exec_run('kong migrations up --v').exit_code == 0

        with run_container(kong_image, environment=kong_env) as kong:
            kong_ip = container_ip(kong)

            def kong_is_listening():
                try:
                    return get('http://{}:8001/signalfx'.format(kong_ip)).status_code == 200
                except RequestException:
                    return False

            assert wait_for(kong_is_listening)

            config = string.Template(dedent('''
            monitors:
              - type: collectd/kong
                host: $host
                port: 8001
                metrics:
                  - metric: connections_handled
                    report: true
            ''')).substitute(host=container_ip(kong))

            with run_agent(config) as [backend, _, _]:
                assert wait_for(p(has_datapoint_with_dim, backend, 'plugin', 'kong')), "Didn't get Kong data point"
