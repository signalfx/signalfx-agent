# Integration Tests

The Go packages have their own tests, but this contains higher-level
integration tests run with [pytest](https://docs.pytest.org/en/latest/).  The
tests need access to the Docker Engine API to start up test services.

These tests run the agent and associated packaging in a more fully functional
environment, along with a fake backend that simulates the SignalFx ingest and
API servers.  

To run all of them in parallel, simply invoke `pytest tests -n auto` from the
root of the repo in the dev image. You can run individual test files by
replacing `tests` with the relative path to the test Python module. You can
also use the `-k` and `-m` flags to pick tests by name or tags, respectively.

A key goal in writing these tests is that they be both fully parallelizable to
minimize run time, and very robust with minimal transient failures due to
timing issues or race conditions that are so prevalent with integration
testing.  Please keep these goals in mind when adding integration tests.

These tests will be run automatically by CircleCI upon each commit.
