import time
from functools import partial as p

import mysql.connector

import pytest
from tests.helpers.agent import Agent
from tests.helpers.assertions import has_datapoint_with_dim, has_datapoint_with_metric_name, tcp_socket_open
from tests.helpers.metadata import Metadata
from tests.helpers.util import container_ip, ensure_always, run_container, wait_for
from tests.helpers.verify import (
    verify,
    verify_expected_is_subset,
    run_agent_verify_included_metrics,
    run_agent_verify_all_metrics,
)

pytestmark = [pytest.mark.docker_container_stats, pytest.mark.monitor_without_endpoints]

ENV = {
    "MYSQL_DATABASE": "testdb",
    "MYSQL_USER": "testuser",
    "MYSQL_PASSWORD": "testpass",
    "MYSQL_ROOT_PASSWORD": "testpass",
}

METADATA = Metadata.from_package("collectd/mysql")


def test_mysql_included():
    with run_container("mysql:5.7", environment=ENV) as mysql:
        host_ip = container_ip(mysql)
        assert wait_for(p(tcp_socket_open, host_ip, 3306), 60), "service didn't start"

        create_db_activity(host_ip)

        run_agent_verify_included_metrics(
            f"""
            monitors:
            - type: collectd/mysql
              host: {host_ip}
              port: 3306
              username: {ENV["MYSQL_USER"]}
              password: {ENV["MYSQL_PASSWORD"]}
              databases:
                - name: {ENV["MYSQL_DATABASE"]}
            """,
            METADATA,
        )


def create_db_activity(host):
    conn = get_mysql_connection(host)
    create_alter_table(conn)
    insert_rows(conn)
    update_rows(conn)
    select_rows(conn)
    delete_rows(conn)
    conn.close()


def get_mysql_connection(host):
    return mysql.connector.connect(
        host=host, user="root", passwd=ENV["MYSQL_ROOT_PASSWORD"], database=ENV["MYSQL_DATABASE"]
    )


def create_alter_table(conn):
    cursor = conn.cursor()
    cursor.execute("CREATE TABLE customers (name VARCHAR(255), address VARCHAR(255))")
    cursor.execute("ALTER TABLE customers ADD COLUMN id INT AUTO_INCREMENT PRIMARY KEY")


def insert_rows(conn):
    cursor = conn.cursor()
    sql = "INSERT INTO customers (name, address) VALUES (%s, %s)"
    val = ("John", "Highway 21")
    cursor.execute(sql, val)
    val = ("Mike", "7881 Circ Dr")
    cursor.execute(sql, val)
    conn.commit()


def update_rows(conn):
    cursor = conn.cursor()
    cursor.execute("UPDATE customers SET address = '7882 Circle Dr' WHERE name = 'Mike'")
    conn.commit()


def select_rows(conn):
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM customers")
    rows = cursor.fetchall()


def delete_rows(conn):
    cursor = conn.cursor()
    cursor.execute("DELETE FROM customers WHERE name = 'John'")
