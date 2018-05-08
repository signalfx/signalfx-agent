RUN_CONTAINER := neo-agent-tmp

.PHONY: check
check: lint vet test

.PHONY: test
test: templates
	go test ./...

.PHONY: vet
vet: templates
	# Only consider it a failure if issues are in non-test files
	! go vet ./... 2>&1 | tee /dev/tty | grep '.go' | grep -v '_test.go'

.PHONY: vetall
vetall: templates
	go vet ./...

.PHONY: lint
lint:
	golint -set_exit_status ./cmd/... ./internal/...

templates:
	scripts/make-templates

.PHONY: image
image:
	./scripts/build

.PHONY: vendor
vendor:
	dep ensure

signalfx-agent: templates
	echo "building SignalFx agent for operating system: $(GOOS)"
	CGO_ENABLED=0 go build \
		-ldflags "-X main.Version=$(AGENT_VERSION) -X main.BuiltTime=$$(date +%FT%T%z)" \
		-o signalfx-agent \
		github.com/signalfx/signalfx-agent/cmd/agent

.PHONY: bundle
bundle:
	BUILD_BUNDLE=true scripts/build

.PHONY: deb-package
deb-%-package:
	packaging/deb/build $*

.PHONY: rpm-package
rpm-%-package:
	packaging/rpm/build $*

.PHONY: attach-image
run-shell:
# Attach to the running container kicked off by `make run-image`.
	docker exec -it $(RUN_CONTAINER) bash

.PHONY: dev-image
dev-image:
	bash -ec "source scripts/common.sh && do_docker_build signalfx-agent-dev latest dev-extras"

.PHONY: debug
debug:
	dlv debug ./cmd/agent

.PHONY: run-dev-image
run-dev-image:
	docker exec -it -e COLUMNS=`tput cols` -e LINES=`tput lines` signalfx-agent-dev /bin/bash -l -i 2>/dev/null || \
	  docker run --rm -it \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		--net host \
		-p 6060:6060 \
		--name signalfx-agent-dev \
		-v $(PWD)/local-etc:/etc/signalfx \
		-v /:/hostfs:ro \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		-v $(PWD):/go/src/github.com/signalfx/signalfx-agent:cached \
		-v $(PWD)/collectd:/usr/src/collectd:cached \
		-v /tmp/scratch:/tmp/scratch \
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

