# Developer's Guide

The agent is built from a single multi-stage Dockerfile. This requires Docker
17.06+.  There is a dev image that can be built for more convenient local
development. Run `make dev-image` to build it and `make run-dev-image` to run
it and attach to a shell inside of it.  Inside this dev image, the agent bundle
is at `/bundle` and the rest of the image contains useful tools for
development, such as a golang build environment.

Within this image, you can build the agent with `make signalfx-agent` and then
run the agent with `./signalfx-agent`.  The code directory will be mounted in
the container at the right place in the Go path so that it can be built with no
extra setup.  There is also an environment variable `SIGNALFX_BUNDLE_DIR` set
to `/bundle` so that the agent knows where to find the bundle when run.

You can put agent config in the `local-etc` dir of this repo and it will be
shared into the container at the default place that the agent looks for config
(`/etc/signalfx`).

## Trivial Commits
If you have a very simple commit that should not require a full CI run, just
put the text `[skip ci]` in the commit message somewhere and CircleCI will not
run for that commit.

## Dependencies

We are using [dep](https://github.com/golang/dep) to manage dependencies.  It
isn't quite GA yet but seems to meet our needs sufficiently.  Vendoring the
Kubernetes client-go requires a bit of hacking in the Gopkg.toml depedencies
but wasn't too bad to get working, despite the fact that they officially don't
recommend using it with dep.

If you add another Go package dependency, you can manually add it to the
[Gopkg.toml](../Gopkg.toml) if you want to specify an exact dependency version,
or you can just use the dependency in your code and run `dep ensure` and it
will take care of figuring out a version that works and adding it to the
`Gopkg.*` files.

## Development in Kubernetes (K8s)

You can use [minikube](https://github.com/kubernetes/minikube) when testing
certain aspects of the K8s observers and monitors, but minikube is limited to a
single node.

If you are a SignalFx employee, we have a private quay.io repository at
`quay.io/signalfx/signalfx-agent-dev` where you can push test images to be
deployed to K8s.  If you are not an employee, quay.io offers free repositories
as long as they are public, so you can make one.

[Helm](https://github.com/kubernetes/helm) makes it easy to deploy the agent as
well as services to monitor on K8s.  There is a Helm values file for
development [in this repo](../deployments/helm-dev-values.yaml]) that will use
the quay.io private repo.

## Running tests

The agent comes with a suite of unit and integration tests which exercise
various components within the agent.  All of these tests must pass for a branch
to be merged into the mainline `master` branch.  Our CircleCI configuration
will automatically run them when a pull request is made but you can run them
manually as follows:

### Go Unit Tests
Simply run `make tests`.  You should add new unit tests for any new modules
that have relatively self-contained functionality that is easy to isolate and
test.

### Integration Tests
These tests run the agent and associated packaging in a more fully functional
environment, along with a fake backend that simulates the SignalFx ingest and
API servers.  

These are all written using Python's pytest and are located in the `tests`
directory.  To run all of them in parallel, simply invoke `pytest tests -n
auto` from the root of the repo in the dev image.  You can select certain tests
to run by either specifying a single python module or by using the `-k` and
`-m` flags.

A key goal in writing these tests is that they be both fully parallelizable to
minimize run time, and very robust with minimal transient failures due to
timing issues or race conditions that are so prevalent with integration
testing.  Please keep these goals in mind when adding integration tests.

### Lint
We require 100% passing rate for the standard [golint
tool](https://github.com/golang/lint), which can be run with `make lint`.

### Vet
We also require 100% passing for [go vet](https://golang.org/cmd/vet/) for
non-test code.  Test code can fail if there is a good reason.
