RUN_CONTAINER := neo-agent-tmp
COLLECTD_VERSION := 5.8.0-sfx0
COLLECTD_COMMIT := 4da1c1cbbe83f881945088a41063fe86d1682ecb
.PHONY: check
check: lint vet test

.PHONY: test
test: templates
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; test }"
else
	CGO_ENABLED=0 go test ./...
endif

.PHONY: vet
vet: templates
	# Only consider it a failure if issues are in non-test files
	! CGO_ENABLED=0 go vet ./... 2>&1 | tee /dev/tty | grep '.go' | grep -v '_test.go'

.PHONY: vetall
vetall: templates
	CGO_ENABLED=0 go vet ./...

.PHONY: lint
lint:
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; lint }"
else
	CGO_ENABLED=0 golint -set_exit_status ./cmd/... ./internal/...
endif

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
	CGO_ENABLED=0 dep ensure
endif


signalfx-agent: templates
	echo "building SignalFx agent for operating system: $(GOOS)"
ifeq ($(OS),Windows_NT)
	powershell "& { . $(CURDIR)/scripts/windows/make.ps1; signalfx-agent $(AGENT_VERSION)}"
else
	CGO_ENABLED=0 go build \
		-ldflags "-X main.Version=$(AGENT_VERSION) -X main.CollectdVersion=$(COLLECTD_VERSION) -X main.BuiltTime=$$(date +%FT%T%z)" \
		-o signalfx-agent \
		github.com/signalfx/signalfx-agent/cmd/agent
endif

.PHONY: bundle
bundle:
	BUILD_BUNDLE=true COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) scripts/build

.PHONY: deb-package
deb-%-package:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) packaging/deb/build $*

.PHONY: rpm-package
rpm-%-package:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) packaging/rpm/build $*

.PHONY: attach-image
run-shell:
# Attach to the running container kicked off by `make run-image`.
	docker exec -it $(RUN_CONTAINER) bash

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
docker_env = -e COLUMNS=`tput cols` -e LINES=`tput lines`
endif

.PHONY: run-dev-image
run-dev-image:
	docker exec -it $(docker_env) signalfx-agent-dev /bin/bash -l -i || \
	  docker run --rm -it \
		$(extra_run_flags) \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		--net host \
		-p 6060:6060 \
		--name signalfx-agent-dev \
		-v $(CURDIR)/local-etc:/etc/signalfx \
		-v $(CURDIR):/go/src/github.com/signalfx/signalfx-agent:cached \
		-v $(CURDIR)/collectd:/usr/src/collectd:cached \
		signalfx-agent-dev /bin/bash

.PHONY: run-integration-tests
run-integration-tests:
	docker run --rm \
		$(extra_run_flags) \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		--net host \
		-v $(CURDIR)/local-etc:/etc/signalfx \
		-v $(CURDIR)/tests:/go/src/github.com/signalfx/signalfx-agent/tests:cached \
		-v $(CURDIR)/collectd:/usr/src/collectd:cached \
		-v $(CURDIR)/test_output:/test_output \
		signalfx-agent-dev \
			pytest \
				-m "not packaging and not installer and not k8s" \
				-n 4 \
				--verbose \
				--exitfirst \
				--html=/test_output/results.html \
				--self-contained-html \
				tests

K8S_VERSION = v1.10.0
.PHONY: run-k8s-tests
run-k8s-tests:
	docker rm -f -v minikube || true
	pytest \
		-m "k8s" \
		-n 4 \
		--verbose \
		--exitfirst \
		--k8s-version=$(K8S_VERSION) \
		--k8s-observers=k8s-api,k8s-kubelet \
		--html=test_output/k8s_results.html \
		--self-contained-html \
		tests || true
	docker rm -f -v minikube

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
