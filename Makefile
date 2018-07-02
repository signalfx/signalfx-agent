RUN_CONTAINER := neo-agent-tmp
COLLECTD_VERSION := 5.8.0-sfx0
COLLECTD_COMMIT := 67fe36b0d5c88054be3b975afc2707db0ecc9022
.PHONY: check
check: lint vet test

.PHONY: test
test: templates
ifeq ($(OS),Windows_NT)
	powershell -Command $$env:CGO_ENABLED=0; go test ./...
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
	CGO_ENABLED=0 golint -set_exit_status ./cmd/... ./internal/...

templates:
ifneq ($(OS),Windows_NT)
	scripts/make-templates
endif

.PHONY: image
image:
	COLLECTD_VERSION=$(COLLECTD_VERSION) COLLECTD_COMMIT=$(COLLECTD_COMMIT) ./scripts/build

.PHONY: vendor
vendor:
	CGO_ENABLED=0 dep ensure

signalfx-agent: templates
	echo "building SignalFx agent for operating system: $(GOOS)"
ifeq ($(OS),Windows_NT)
	powershell -Command $$env:CGO_ENABLED=0; go build -ldflags \"-X main.Version=$(AGENT_VERSION) -X main.BuiltTime=$$(Get-Date  -UFormat \"%Y-%m-%dT%T%Z\")\" -o signalfx-agent.exe github.com/signalfx/signalfx-agent/cmd/agent
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
	powershell -Command . $(CURDIR)\scripts\windows\common.ps1; do_docker_build signalfx-agent-dev latest dev-extras
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

.PHONY: docs
docs:
	bash -c "rm -f docs/{observers,monitors}/*"
	scripts/docs/make-docs

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
	echo $(COLLECTD_VERSION)

.PHONY: collectd-commit
collectd-commit:
	echo $(COLLECTD_COMMIT)
