# Developer's Guide

The agent is built from a single multi-stage Dockerfile. This requires Docker
17.06+.  There is a dev image that can be built for more convenient local
development. To use it first do the following:

```sh
$ make dev-image
$ # The build will take a while...
$ scripts/prime-vendor-dir  # Copy out the vendor dir from the dev image
$ mkdir local-etc && cp packaging/etc/agent.yaml local-etc/  # Get a basic operating config file in place
$ make run-dev-image
```

If all went well, you will now be attached to a shell inside of the dev image
at the Go package dir where you can compile and run the agent.

Inside this image the agent "bundle" (which basically means collectd and all of
its dependencies) is at `/bundle`, and the rest of the dev image contains
useful tools for development, such as a golang build environment.

Within this image, you can build the agent with `make signalfx-agent` and then
run the agent with `./signalfx-agent`.  The code directory will be mounted in
the container at the right place in the Go path so that it can be built with no
extra setup.  There is also an environment variable `SIGNALFX_BUNDLE_DIR` set
to `/bundle` so that the agent knows where to find the bundle when run.  The
agent binary itself is statically compiled, so it has no external library
dependencies once built.

You can put agent config in the `local-etc` dir of this repo and it will be
shared into the container at the default place that the agent looks for config
(`/etc/signalfx`).  The `local-etc` dir is ignored by git.

## Making the Final Docker Image
To make the final Docker image without all of the development tools, just run
`make image` (either in or outside of the dev image) and it will make a new
agent image with the name `quay.io/signalfx/signalfx-agent-dev:<agent
version>`.  The agent version will be automatically determined from the git
repo status, but can be overridden with the `AGENT_VERSION` envvar.  The image
name itself can be overridden with the `AGENT_IMAGE_NAME` envvar.

## Making the Standalone Bundle
To make the standalone `.tar.gz` bundle, simply run `make bundle` (either in or
outside of the dev image).  It will dump a file with the name
`signalfx-agent-<agent version>.tar.gz` in the current directory.  You can
override the agent version used with the `AGENT_VERSION` envvar, otherwise it
will be automatically inferred from the git repo.

## Contributing
If you are a SignalFx employee you should make commits to a branch off of the
main code repository at https://github.com/signalfx/signalfx-agent and make a
pull request back to the master branch.  If you are not an employee, simply
fork that repository and make pull requests back to our repo's master branch.
We welcome any enhancements you might have -- we will try to respond to all
issues and pull requests quickly.

### Trivial Commits
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
These are all written using Python's pytest and are located in the [tests
directory](https://github.com/signalfx/signalfx-agent/tree/master/tests).  See
there for more information.

### Lint
We require 100% passing rate for the standard [golint
tool](https://github.com/golang/lint), which can be run with `make lint`.

### Vet
We also require 100% passing for [go vet](https://golang.org/cmd/vet/) for
non-test code.  Test code can fail if there is a good reason.
