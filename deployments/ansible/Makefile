
.PHONY: dev-image
dev-image:
	docker build -t signalfx-agent-ansible-dev .

.PHONY: run-dev-image
run-dev-image:
	docker exec -it signalfx-agent-ansible-dev 2>/dev/null || \
	docker run \
		--rm \
		--cap-add DAC_READ_SEARCH \
		--cap-add SYS_PTRACE \
		--name signalfx-agent-ansible-dev \
		-it \
		signalfx-agent-ansible-dev \
                bash
