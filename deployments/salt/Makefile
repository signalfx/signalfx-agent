.PHONY: dev-image
dev-image:
	docker build -t signalfx-agent-salt-dev .

.PHONY: run-dev-image
run-dev-image:
	docker exec -it signalfx-agent-salt-dev 2>/dev/null || \
	docker run \
		--rm \
		--name signalfx-agent-salt-dev \
		-it \
		signalfx-agent-salt-dev \
		bash

.PHONY: test
test:
	salt-lint -v /srv/salt/signalfx-agent/*.sls
	salt-call --local state.show_sls signalfx-agent
	salt-call --local pillar.items
