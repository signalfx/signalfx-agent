# Integration Tests

The Go packages have their own tests, but this contains higher-level
integration tests run with [pytest](https://docs.pytest.org/en/latest/).  The
tests need access to the Docker Engine API to start up test services.

These tests run the agent and associated packaging in a more fully functional
environment, along with a fake backend that simulates the SignalFx ingest and
API servers.  

To run all of them in parallel, simply invoke `pytest tests -n auto` from the
root of the repo in the dev-image (see note below for kubernetes
limitations). You can run individual test files by replacing `tests` with the 
relative path to the test Python module. You can also use the `-k` and `-m` 
flags to pick tests by name or tags, respectively.

A key goal in writing these tests is that they be both fully parallelizable to
minimize run time, and very robust with minimal transient failures due to
timing issues or race conditions that are so prevalent with integration
testing.  Please keep these goals in mind when adding integration tests.

These tests will be run automatically by CircleCI upon each commit.

As a quickstart for running the tests locally, the following commands will
build the dev-image, and run the integration and kubernetes tests within the
dev-image with the default markers and options (check the respective targets in
[Makefile](../Makefile) for the default markers and options):
```
% cd <root_of_cloned_repo>
% make dev-image  # build the dev-image (signalfx-agent-dev:latest)
% make run-dev-image  # start the dev-image container with interactive shell
%% make run-integration-tests
%% make run-k8s-tests
```

The `make run-integration-tests` and `make run-k8s-tests` targets should only
run only within the dev-image container.  To override the default markers, set
the `MARKERS` environment variable to the desired list of markers for pytest to
collect. For example, the following command will run all tests with the `k8s`
marker:
```
%% MARKERS="k8s" make run-k8s-tests
```

By default, the kubernetes tests will build and test the agent image from the
local source (i.e. the image built by `make image`) and tag it as
`signalfx-agent:k8s-test`.  Alternatively, you can specify the name of a
pre-built agent image with the `K8S_SFX_AGENT=NAME:TAG` environment variable
(if the image does not exist in the local registry, the test will try to pull
the image from the remote registry). Also, the current default kubernetes
cluster version that is deployed within minikube is `v1.13.0`.  To deploy and
test with a different version, specify the version with the `K8S_VERSION=vX.Y.Z`
environment variable (`latest` is also acceptable and will deploy the latest
kubernetes version supported by minikube).  For example, the following command
will deploy a `v1.8.0` kubernetes cluster within the minikube container and run
tests with the released `quay.io/signalfx/signalfx-agent:4.0.0` agent image:
```
%% K8S_VERSION=v1.8.0 K8S_SFX_AGENT=quay.io/signalfx/signalfx-agent:4.0.0 make run-k8s-tests
```

**Note:** Due to known limitations of the xdist pytest plugin (e.g. the
`-n auto` option), fixtures cannot be shared across workers.  In order to prevent
starting multiple minikube containers (fixtures) for each worker when running the
kubernetes tests in parallel (e.g. `make run-k8s-tests`), the minikube container
will be started once for all workers, but it will not be automatically removed
when the tests complete since there is currently no way of knowing which test
will be last to teardown the fixture.  As a workaround, the `make run-k8s-tests`
command will remove the minikube container if all tests pass, but will leave the
container running for debugging purposes if there are any failures.  Any running
containers named `minikube` will automatically be removed during the kubernetes
test setup, and a new container will be started.

Run `pytest --help` from the root of the repo in the dev-image and see the
"custom options" section of the output for more details and other available
options.

## Formatting

Test code should be formatted automatically with
[Black](https://pypi.org/project/black/) (see
[./requirements.txt](./requirements.txt) for the exact version) and should also
pass Pylint.  We use a line length of 120.

Running `make lint-pytest` within the dev-image will format the tests with Black
and execute Pylint.  Resolve any issues before committing/pushing the changes.
