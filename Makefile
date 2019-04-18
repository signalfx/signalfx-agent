COLLECTD_VERSION := 5.8.0-sfx0
COLLECTD_COMMIT := 4da1c1cbbe83f881945088a41063fe86d1682ecb
BUILD_TIME ?= $$(date +%FT%T%z)
ifeq ($(OS),Windows_NT)
MONITOR_CODE_GEN := monitor-code-gen.exe
else
MONITOR_CODE_GEN := ./monitor-code-gen
endif
NUM_CORES ?= $(shell getconf _NPROCESSORS_ONLN)

.PHONY: check
check: lint vet test

.PHONY: compileDeps
compileDeps: templates code-gen internal/core/common/constants/versions.go

.PHONY: code-gen
code-gen: $(MONITOR_CODE_GEN)

$(MONITOR_CODE_GEN): $(wildcard cmd/monitorcodegen/*.go)
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; monitor-code-gen }"
else
	go build -mod vendor -o $@ ./cmd/monitorcodegen
endif

.PHONY: test
test: compileDeps
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; test }"
else
	CGO_ENABLED=0 go test -mod vendor -p $(NUM_CORES) ./...
endif

.PHONY: vet
vet: compileDeps
	# Only consider it a failure if issues are in non-test files
	! CGO_ENABLED=0 go vet -mod vendor ./... 2>&1 | tee /dev/tty | grep '.go' | grep -v '_test.go'

.PHONY: vetall
vetall: compileDeps
	CGO_ENABLED=0 go vet -mod vendor ./...

.PHONY: lint
lint:
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; lint }"
else
	CGO_ENABLED=0 golint -set_exit_status ./cmd/... ./internal/...
endif

.PHONY: gofmt
gofmt:
	CGO_ENABLED=0 go fmt ./...

templates:
ifneq ($(OS),Windows_NT)
	scripts/make-templates
endif

.PHONY: image
image:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) ./scripts/build

.PHONY: vendor
vendor:
ifeq ($(OS), Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; vendor }"
else
	go mod tidy && go mod vendor
endif

internal/core/common/constants/versions.go: FORCE
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; versions_go }"
else
	AGENT_VERSION=$(AGENT_VERSION) COLLECTD_VERSION=$(COLLECTD_VERSION) BUILD_TIME=$(BUILD_TIME) scripts/make-versions
endif

signalfx-agent: compileDeps
	echo "building SignalFx agent for operating system: $(GOOS)"
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; signalfx-agent $(AGENT_VERSION)}"
else
	CGO_ENABLED=0 go build \
		-mod vendor \
		-o signalfx-agent \
		./cmd/agent
endif

.PHONY: bundle
bundle:
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; bundle $(COLLECTD_COMMIT)}"
else
	BUILD_BUNDLE=true COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) scripts/build
endif

.PHONY: deb-package
deb-%-package:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) packaging/deb/build $*

.PHONY: rpm-package
rpm-%-package:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) packaging/rpm/build $*

.PHONY: dev-image
dev-image:
ifeq ($(OS),Windows_NT)
	powershell -Command "& { . $(CURDIR)\scripts\windows\common.ps1; do_docker_build signalfx-agent-dev latest dev-extras }"
else
	bash -ec "COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) && source scripts/common.sh && do_docker_build signalfx-agent-dev latest dev-extras"
endif

.PHONY: debug
debug:
	dlv debug ./cmd/agent

ifneq ($(OS),Windows_NT)
extra_run_flags = -v /:/hostfs:ro -v /var/run/docker.sock:/var/run/docker.sock:ro -v /tmp/scratch:/tmp/scratch
ifeq ($(shell uname -s),Linux)
extra_run_flags += -v /var/run/dbus/system_bus_socket:/var/run/dbus/system_bus_socket:ro
endif
docker_env = -e COLUMNS=`tput cols` -e LINES=`tput lines`
endif

.PHONY: run-dev-image
run-dev-image:
	docker exec -it $(docker_env) signalfx-agent-dev /bin/bash -l -i || \
	  docker run --rm -it \
		--privileged \
		$(extra_run_flags) \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		-p 6060:6060 \
		-p 9080:9080 \
		-p 8095:8095 \
		--name signalfx-agent-dev \
		-v $(CURDIR)/local-etc:/etc/signalfx \
		-v $(CURDIR):/usr/src/signalfx-agent:delegated \
		-v $(CURDIR)/collectd:/usr/src/collectd:delegated \
		-v $(CURDIR)/tmp/pprof:/tmp/pprof \
		signalfx-agent-dev /bin/bash

.PHONY: run-dev-image-sync
run-dev-image-sync:
	docker exec -it $(docker_env) signalfx-agent-dev-sync /bin/bash -l -i || \
	  docker run --rm -it \
		$(extra_run_flags) \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		-p 6061:6060 \
		-p 9081:9080 \
		-p 8096:8095 \
		--name signalfx-agent-dev-sync \
		-v signalfx-agent-sync:/usr/src/signalfx-agent:nocopy \
		-v $(CURDIR)/local-etc:/etc/signalfx \
		-v $(CURDIR)/tmp/pprof:/tmp/pprof \
		signalfx-agent-dev /bin/bash

.PHONY: run-dev-image-commands
run-dev-image-commands:
	docker exec -t $(docker_env) signalfx-agent-dev /bin/bash -c '$(RUN_DEV_COMMANDS)'

.PHONY: run-integration-tests
run-integration-tests: MARKERS ?= not packaging and not bundle and not installer and not kubernetes and not windows_only and not deployment and not perf_test
run-integration-tests:
	AGENT_BIN=/bundle/bin/signalfx-agent \
	pytest \
		-m "$(MARKERS)" \
		-n auto \
		--verbose \
		--html=test_output/results.html \
		--self-contained-html \
		tests

.PHONY: run-k8s-tests
run-k8s-tests: MARKERS ?= (kubernetes or helm) and not collectd
run-k8s-tests:
	pytest \
		-m "$(MARKERS)" \
		-n auto \
		--verbose \
		--exitfirst \
		--html=test_output/k8s_results.html \
		--self-contained-html \
		tests || \
	(docker ps | grep -q minikube && echo "minikube container is left running for debugging purposes"; return 1) && \
	docker rm -fv minikube

.PHONY: docs
docs:
	bash -c "rm -f docs/{observers,monitors}/*"
	scripts/docs/make-docs

.PHONY: stage-cache
stage-cache:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) scripts/tag-and-push-targets

.PHONY: product-docs
product-docs:
	scripts/docs/to-product-docs

.PHONY: integrations-repo
integrations-repo:
	scripts/docs/to-integrations-repo

.PHONY: chef-%
chef-%:
	$(MAKE) -C deployments/chef $*

.PHONY: puppet-%
puppet-%:
	$(MAKE) -C deployments/puppet $*

.PHONY: collectd-version
collectd-version:
	@echo ${COLLECTD_VERSION}

.PHONY: collectd-commit
collectd-commit:
	@echo ${COLLECTD_COMMIT}

.PHONY: lint-pytest
lint-pytest:
	scripts/lint-pytest

.PHONY: lint-python
lint-python:
	scripts/lint-python

.PHONY: devstack
devstack:
	scripts/make-devstack-image

.PHONY: run-devstack
run-devstack:
	scripts/run-devstack-image

.PHONY: run-chef-tests
run-chef-tests:
	pytest -v -n auto -m chef --html=test_output/chef_results.html --self-contained-html tests/deployments

K8S_VERSION ?= latest
.PHONY: run-minikube
run-minikube:
	python -c 'from tests.helpers.kubernetes.minikube import Minikube; mk = Minikube(); mk.deploy("$(K8S_VERSION)")' && \
	docker exec -it $(docker_env) minikube /bin/bash

FORCE:
