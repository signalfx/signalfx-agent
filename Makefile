RUN_CONTAINER := neo-agent-tmp

go_pkgs := $(shell glide novendor)

.PHONY: check
check: lint vet test

.PHONY: test
test:
	@if [ "$(go_pkgs)" = "" ]; then\
		echo 'Go packages could not be determined' && exit 1;\
	else\
		go test $(go_pkgs);\
	fi

.PHONY: vet
vet:
	go vet $(go_pkgs)

.PHONY: lint
lint:
	golint -set_exit_status $(go_pkgs)

.PHONY: collectd
collectd:
	./scripts/build-collectd.sh

templates:
	scripts/make-templates

.PHONY: image
image:
	./scripts/build.sh

image-debug:
	DEBUG=true ./scripts/build.sh

.PHONY: vendor
vendor:
	glide update --strip-vendor
	sed -i '' -e 's/Sirupsen/sirupsen/' $$(grep -lR Sirupsen vendor) || true

.PHONY: run-image
run-image:
# Run a pre-built image locally. When the agent terminates or you ctrl-c the container is removed.
# Setup: cp -r etc local-etc and make any changes necessary to agent.yaml.
	docker run -it --rm \
		--name $(RUN_CONTAINER) \
		-e SFX_ACCESS_TOKEN=$(SFX_ACCESS_TOKEN) \
		--privileged \
		--net host \
		-v $(PWD)/local-etc:/etc/signalfx \
		-v /:/hostfs:ro \
		-v /etc/hostname:/mnt/hostname:ro \
		-v /etc:/mnt/etc:ro \
		-v /proc:/mnt/proc:ro \
		-v /var/run/docker.sock:/var/run/docker.sock \
		quay.io/signalfuse/signalfx-agent:$(USER)

.PHONY: attach-image
run-shell:
# Attach to the running container kicked off by `make run-image`.
	docker exec -it $(RUN_CONTAINER) bash

.PHONY: dev-image
dev-image:
	scripts/make-dev-image

.PHONY: run-dev-image
run-dev-image:
	docker run --rm -it \
		--privileged \
		--net host \
		-v $(PWD)/local-etc:/etc/signalfx \
		-v /:/hostfs:ro \
		-v /etc:/mnt/etc:ro \
		-v /proc:/mnt/proc:ro \
		-v $(PWD):/go/src/github.com/signalfx/neo-agent \
		-v /var/run/docker.sock:/var/run/docker.sock \
		neoagent-dev /bin/bash
