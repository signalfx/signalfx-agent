COLLECTD_VERSION := 5.8.0-sfx0
COLLECTD_COMMIT := 4da1c1cbbe83f881945088a41063fe86d1682ecb
BUILD_TIME ?= $$(date +%FT%T%z)
NUM_CORES ?= $(shell getconf _NPROCESSORS_ONLN)

.PHONY: clean
clean:
	rm -f pkg/core/constants/versions.go
	find pkg/monitors -name "genmetadata.go" -delete
	find pkg/monitors -name "template.go" -delete
	rm -f pkg/monitors/collectd/collectd.conf.go
	rm -f pkg/monitors/zcodegen/monitorcodegen
	rm -f signalfx-agent

.PHONY: check
check: lint vet test

.PHONY: test
test:
	go generate ./...
	CGO_ENABLED=0 go test -p $(NUM_CORES) ./...

.PHONY: vet
vet:
	go generate ./...
	# Only consider it a failure if issues are in non-test files
	! CGO_ENABLED=0 go vet ./... 2>&1 | tee /dev/tty | grep '.go' | grep -v '_test.go'

.PHONY: vetall
vetall:
	go generate ./...
	CGO_ENABLED=0 go vet ./...

.PHONY: lint
lint:
	go generate ./...
	@echo 'Linting LINUX code'
	CGO_ENABLED=0 GOGC=40 golangci-lint run --deadline 5m
	@echo 'Linting WINDOWS code'
	GOOS=windows CGO_ENABLED=0 GOGC=40 golangci-lint run --deadline 5m

.PHONY: gofmt
gofmt:
	CGO_ENABLED=0 go fmt ./...

.PHONY: image
image:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) ./scripts/build

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: signalfx-agent
signalfx-agent:
	go generate ./...
	echo "building SignalFx agent for operating system: $(GOOS)"
	CGO_ENABLED=0 go build \
		-o signalfx-agent \
		./cmd/agent

.PHONY: set-caps
set-caps:
	sudo setcap CAP_SYS_PTRACE,CAP_DAC_READ_SEARCH=+eip ./signalfx-agent

.PHONY: bundle
bundle:
ifeq ($(OS),Windows_NT)
	powershell $(CURDIR)/scripts/windows/make.ps1 bundle $(COLLECTD_COMMIT)
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
	bash -ec "COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) && source scripts/common.sh && do_docker_build signalfx-agent-dev latest dev-extras"

.PHONY: debug
debug:
	dlv debug ./cmd/agent

ifdef dbus
# Useful if testing the collectd/systemd monitor
dbus_run_flags = --privileged -v /var/run/dbus/system_bus_socket:/var/run/dbus/system_bus_socket:ro
endif

ifneq ($(OS),Windows_NT)
extra_run_flags = -v /:/hostfs:ro -v /var/run/docker.sock:/var/run/docker.sock:ro -v /tmp/scratch:/tmp/scratch
docker_env = -e COLUMNS=`tput cols` -e LINES=`tput lines`
endif

.PHONY: run-dev-image
run-dev-image:
	docker exec -it $(docker_env) signalfx-agent-dev /bin/bash -l -i || \
	  docker run --rm -it \
		$(dbus_run_flags) $(extra_run_flags) \
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

.PHONY: run-integration-tests
run-integration-tests: MARKERS ?= integration
run-integration-tests:
	pytest \
		-m "$(MARKERS)" \
		-n auto \
		--verbose \
		--html=test_output/results.html \
		--self-contained-html \
		tests

.PHONY: run-k8s-tests
run-k8s-tests: MARKERS ?= (kubernetes or helm) and not collectd
run-k8s-tests: run-minikube push-minikube-agent
	scripts/get-kubectl
	pytest \
		-m "$(MARKERS)" \
		-n auto \
		--verbose \
		--html=test_output/k8s_results.html \
		--self-contained-html \
		tests

K8S_VERSION ?= latest
MINIKUBE_VERSION ?= v1.4.0

.PHONY: run-minikube
run-minikube:
	docker build \
		-t minikube:$(MINIKUBE_VERSION) \
		--build-arg 'MINIKUBE_VERSION=$(MINIKUBE_VERSION)' \
		test-services/minikube

	docker rm -fv minikube 2>/dev/null || true
	docker run -d \
		--name minikube \
		--privileged \
		-e K8S_VERSION=$(K8S_VERSION) \
		-e TIMEOUT=$(MINIKUBE_TIMEOUT) \
		-p 5000:5000 \
		minikube:$(MINIKUBE_VERSION)

	docker exec minikube start-minikube.sh

	echo "Minikube is now running. Push up an agent image to localhost:5000/signalfx-agent:latest or run 'make push-minikube-agent'"

.PHONY: push-minikube-agent
push-minikube-agent:
	PUSH=yes \
	  AGENT_IMAGE_NAME=localhost:5000/signalfx-agent \
	  AGENT_VERSION=latest \
	  SKIP_BUILD_PULL=yes \
	  $(MAKE) image

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

.PHONY: check-links
check-links:
	docker build -t check-links scripts/docs/check-links
	docker run --rm -v $(CURDIR):/usr/src/signalfx-agent:ro check-links

.PHONY: dependency-check
dependency-check: BUNDLE_PATH ?= signalfx-agent-$(shell ./scripts/current-version).tar.gz
dependency-check:
	./scripts/dependency-check/run.sh $(BUNDLE_PATH)

FORCE:
