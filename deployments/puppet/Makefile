
.PHONY: dev-image
dev-image:
	docker build -t signalfx-agent-puppet-dev .

.PHONY: run-dev-image
run-dev-image:
	docker exec -it signalfx-agent-puppet-dev bash 2>/dev/null || \
	docker run \
		--rm \
		--name signalfx-agent-puppet-dev \
		-it \
		-v $(CURDIR):/etc/puppetlabs/code/modules/signalfx_agent \
		signalfx-agent-puppet-dev \
		bash

.PHONY: release
release:
	./release
