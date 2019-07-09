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

## Quickstart
The following commands will build the dev-image, and run the integration and
Kubernetes tests within the dev-image with the default markers and options
(check the respective targets in [Makefile](../Makefile) for the default
markers and options):

```sh
$ cd <root_of_cloned_repo>
$ make dev-image  # build the dev-image (signalfx-agent-dev:latest)
$ make run-dev-image  # start the dev-image container with interactive shell
$$ make run-integration-tests
$$ make run-k8s-tests
```

The `make run-integration-tests` and `make run-k8s-tests` targets should only
run only within the dev-image container.  To override the default markers, set
the `MARKERS` environment variable to the desired list of markers for pytest to
collect. For example, the following command will run all tests with the `kubernetes`
marker:
```
%% MARKERS="kubernetes" make run-k8s-tests
```

## Kubernetes Environment
The Kubernetes tests depend upon a running cluster that can pull the agent
image that you want to test.  By default, the tests will look for a container
in the local docker daemon called `minikube`.  If that container exists, it
will use the minikube cluster running inside of that container.  It will then
also by default use an agent image called
`minikube-test:5000/signalfx-agent:latest`.  You can start minikube and push
this image into a registry running within minikube by running the following:

```sh
$ make run-minikube
$ make push-minikube-agent  # Requires sudo to update /etc/hosts with the minikube registry hostname
```

You can specify another image with the `--agent-image-name` flag to pytest.

By default, the minikube that gets run with `make run-minikube` will use the
latest available version of Kubernetes.  To test with a different version,
specify the version with the `K8S_VERSION=vX.Y.Z` environment variable
(`latest` is also acceptable and will deploy the latest kubernetes version
supported by minikube).

You can also use any existing Kubernetes cluster, as long as you can access it
with kubectl from the same host you are running the tests on.

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
