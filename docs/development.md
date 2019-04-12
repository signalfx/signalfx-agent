# Developer's Guide

The agent is built from a single multi-stage Dockerfile. This requires Docker
17.06+.  There is a dev image that can be built for more convenient local
development. To use it first do the following:

```sh
$ make dev-image
$ # The build will take a while...
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

## Development Practices

 - Try and do logging at the highest level of the component you are work on as
   possible.  In the lower-level components, return errors (appropriately
   wrapped with `fmt.Errorf("context of error: %v", err)` to provide context)
   and only at the highest level where it doesn't make sense to return errors
   any more, should you log the error.  E.g. for a monitor, the most approrpriate
   place to do logging is in the function that gets called on an interval.

 - Try and minimize memory allocations as much as possible.  Allocations result
   in higher garbage collection CPU usage.  Don't go crazy on trying to avoid
   allocations in every case where it's possible, but be aware of it.  If
   your code is signficantly harder to understand then it probably isn't worth
   doing unless profiling shows a large benefit.

## Profiling the agent

You can profile the agent with the
[pprof](https://blog.golang.org/profiling-go-programs) tool from Go.  To enable
a profiling HTTP endpoint in the agent, set `profiling: true` in the agent
config.  Then you can hit various endpoints on
`http://localhost:6060/debug/pprof/*` ([where `*` is various profiles
documented here](https://golang.org/pkg/net/http/pprof/)).

## Improve build times on Mac
When developing on Mac and building in a Docker Linux container the source directory is shared using Docker volumes. It is relatively slow and increases build times. A quicker method (2-3x faster) is to do syncing of files to the Docker VM so that file access is in the same host as the Linux container. [docker-sync](http://docker-sync.io) will do this automatically once setup.

Once installed (see below) you run:

```sh
$ docker-sync start
```

and use the target `run-dev-image-sync`. Everything else is the same as described in the previous section.

### Installing docker-sync
Install the `docker-sync` gem:

```sh
$ gem install --user-install docker-sync
```

Make sure the user gems bin directory is in your `PATH`, for example:

```sh
export PATH="${PATH}:$(ruby -r rubygems -e 'puts Gem.user_dir')/bin"
```

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
override the agent version used with the `AGENT_VERSION` envvar; otherwise, the
version will be automatically inferred from the git repo.

## Contributing
If you are a SignalFx employee, you should make commits to a branch off of the
main code repository at https://github.com/signalfx/signalfx-agent and make a
pull request back to the master branch.  If you are not an employee, simply
fork that repository and make pull requests back to our repo's master branch.
We welcome any enhancements you might have, and will try to respond to all
issues and pull requests quickly.

### Trivial Commits
If you have a very simple commit that should not require a full CI run, just
put the text `[skip ci]` in the commit message somewhere and CircleCI will not
run for that commit.

## Go Dependencies

We are using [Go modules](https://github.com/golang/go/wiki/Modules) to manage
dependencies.

We commit all of our go dependencies to the git repository.  This results in a
significantly larger repository but makes the agent build more self-contained
and consistent.

If you add another Go package dependency, you can just run `go get
<package>@<optional_version>`.  Then run `go mod tidy && go mod vendor` and
commit the new vendored source to the repository in the same commit that
depends on the new dependencies.


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
development [in this repo](../deployments/k8s/helm-dev-values.yaml) that will use
the quay.io private repo.

## Running tests

The agent comes with a suite of unit and integration tests that exercise
various components within the agent.  All of these tests must pass for a branch
to be merged into the mainline `master` branch.  Our CircleCI configuration
will automatically run them when a pull request is made, but you can run them
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

### ARM targets

Building and testing for Linux targets with ARM processors can be done on
any ARM environment. Amazon EC2 instances running Amazon Linux 2 for arm64
are known to support the build and test process.

### Windows

We develop on a [VirtualBox](https://www.virtualbox.org/) Windows Server 2008
[Vagrant](https://www.vagrantup.com/).  You might want to develop on Windows
Server 2012+ if you're using the evaluation boxes because the Windows Server
2008 evaluation only has a 10 day trial that can be renewed up to 5 times.
To renew the Windows Server 2008 evaluation you must manually reset the
activation period by using the slmgr.vbs script from the command prompt and
restart the vm.

    slmgr.vbs –rearm

#### Base Box

If you have a valid Windows Vagrant base box,
set the box name in the windows [Vagrant File](./scripts/windows/Vagrantfile).

If you do not have a base box, the makefile target `win-vagrant-base-box` will
checkout the [Windows Boxcutter Project](https://github.com/boxcutter/windows) and build
the Windows Server base box image using the evaluation copy of Windows.

Please ensure that the requisites for Boxcutter are satisfied, including the
installation of [Packer](https://www.packer.io/),
[VirtualBox](https://www.virtualbox.org/), and [Vagrant](https://www.vagrantup.com/).

#### Make File Targets

For convenience the Makefile in the `scripts/windows/vagrant` directory of this project has the following targets:

| Target | Description | Example |
| ------ | ----------- | ------- |
| `win-vagrant-base-box` | Builds a base box using the [Windows Boxcutter Project](https://github.com/boxcutter/windows) | `make win-vagrant-base-box` |
| `win-vagrant-up` | Alias for `vagrant up` that will start and provision the vagrant if necessary. You should be presented with virtualbox vm GUI window when complete. | `make win-vagrant-up` |
| `win-vagrant-destroy` | Alias for `vagrant destroy` that will destroy the vagrant | `make win-vagrant-destroy` |
| `win-vagrant-suspend` | Alias for `vagrant suspend` that will suspend the vagrant | `make win-vagrant-suspend` |
| `win-vagrant-provision` | Alias for `vagrant provision` that will suspend the vagrant | `make win-vagrant-provision` |

By default the makefile uses Windows Server 2008.  If you want to override this, set the environment variable `WINDOWS_VER` to choose a different version.

The following values are supported for `WINDOWS_VER`

| Value | Windows Version | Vagrant Base Box Name | Virtual Box VM Name |
| ----- | --------------- | --------------------- | ------------------- |
| server_2008 | Windows Server 2008 r2 | eval-win2008r2-standard-ssh | Windows_Server_2008_SignalFx_Agent |
| server_2012 | Windows Server 2012 r2 | eval-win2012r2-standard-ssh | Windows_Server_2012_SignalFx_Agent |
| server_2016 | Windows Server 2016 | eval-win2016-standard-ssh | Windows_Server_2016_SignalFx_Agent |

#### Building and Starting the Vagrant

The following snippet will create the vagrant base box, start the vagrant, provision, suspend, and destroy it.

    $ cd $GOPATH/src/github.com/signalfx/signalfx-agent/scripts/windows/vagrant
    $ WIN_VER=server_2008 make win-vagrant-base-box
      ...
    $ WIN_VER=server_2008 make win-vagrant-up
      ...
    $ WIN_VER=server_2008 make win-vagrant-suspend
      ...
    $ WIN_VER=server_2008 make win-vagrant-destroy

`win-vagrant-base-box`, `win-vagrant-up`, and `win-vagrant-provision`
can take a significant amount of time to complete and depend on the
characteristics of your host environment.

#### Software Provisioned in the Vagrant

The vagrant will be provisioned with:
* [Chocolatey](https://chocolatey.org/)
* [Make](https://chocolatey.org/packages/make)
* [Go Lang](https://chocolatey.org/packages/golang)
* [Python3](https://chocolatey.org/packages/python)
* [Dep](https://github.com/golang/dep)
* [git](https://chocolatey.org/packages/git)
* [git credential manager for windows](https://chocolatey.org/packages/Git-Credential-Manager-for-Windows)
* [Visual Studio Code](https://chocolatey.org/packages/VisualStudioCode)
* [Jetbrains Goland](https://chocolatey.org/packages/goland)
* [Jetrains Pycharm](https://chocolatey.org/packages/Pycharm)
* [Firefox](https://chocolatey.org/packages/Firefox)

#### Host Files Included in the Vagrant

This github project `github.com/signalfx/signalfx-agent` will be mapped as a synced directory
to `C:\Users\vagrant\signalfx-agent`.

#### Building For Windows

The vagrant box should have enough dependencies installed that you can build the agent bundle.  To do this navigate to the project in the GOPATH.

    $ cd C:\Users\vagrant\signalfx-agent

    $ & { . ./scripts/windows/make.ps1; bundle }
