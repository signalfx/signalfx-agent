RUN_CONTAINER := neo-agent-tmp

.PHONY: check
check: lint vet test

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golint -set_exit_status utils observers monitors core neotest

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
	dep ensure

signalfx-agent: templates
	go build \
		-ldflags "-X main.Version=$$(./VERSIONS agent_version) -X main.CollectdVersion=$$(./VERSIONS collectd_version) -X main.BuiltTime=$$(date +%FT%T%z)" \
		-i -o signalfx-agent \
		github.com/signalfx/neo-agent

.PHONY: attach-image
run-shell:
# Attach to the running container kicked off by `make run-image`.
	docker exec -it $(RUN_CONTAINER) bash

.PHONY: dev-image
dev-image:
	scripts/make-dev-image

.PHONY: run-dev-image
run-dev-image:
	docker exec -it signalfx-agent-dev bash 2>/dev/null || docker run --rm -it \
		--privileged \
		-p 6060:6060 \
		--name signalfx-agent-dev \
		-v $(PWD)/local-etc:/etc/signalfx \
		-v /:/hostfs:ro \
		-v /etc:/mnt/etc:ro \
		-v /proc:/mnt/proc:ro \
		-v $(PWD):/go/src/github.com/signalfx/neo-agent \
		-v $(PWD)/collectd:/usr/src/collectd \
		-v /var/run/docker.sock:/var/run/docker.sock \
		signalfx-agent-dev /bin/bash
