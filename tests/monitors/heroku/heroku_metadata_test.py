"""
Integration tests for the Heroku metadata monitor
"""
from functools import partial as p

import pytest

from tests.helpers.agent import Agent
from tests.helpers.assertions import has_all_dim_props
from tests.helpers.util import wait_for

HEROKU_ENV = {
    "HEROKU_DYNO_ID": "01234567-89ab-cdef-0123-456789abcdef",
    "HEROKU_RELEASE_VERSION": "v2",
    "HEROKU_SLUG_COMMIT": "df0b51be05db4c15855911366c345c4995139dbe",
    "HEROKU_RELEASE_CREATED_AT": "2019-12-19T18:47:18Z",
    "HEROKU_APP_ID": "5063c52b-8f67-4641-ba8e-2a512dc5cddb",
    "HEROKU_APP_NAME": "myapp",
}


def test_heroku_dyno_metadata():
    with Agent.run(
        """
        monitors:
        - type: heroku-metadata
        """,
        extra_env=HEROKU_ENV,
    ) as agent:
        assert wait_for(
            p(
                has_all_dim_props,
                agent.fake_services,
                dim_name="dyno_id",
                dim_value=HEROKU_ENV["HEROKU_DYNO_ID"],
                props={
                    "heroku_release_version": HEROKU_ENV["HEROKU_RELEASE_VERSION"],
                    "heroku_app_name": HEROKU_ENV["HEROKU_APP_NAME"],
                    "heroku_slug_commit": HEROKU_ENV["HEROKU_SLUG_COMMIT"],
                    "heroku_release_creation_timestamp": HEROKU_ENV["HEROKU_RELEASE_CREATED_AT"],
                    "heroku_app_id": HEROKU_ENV["HEROKU_APP_ID"],
                },
            )
        ), "Didn't get dyno properties"
