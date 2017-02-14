RUN_CONTAINER := neo-agent-tmp

.PHONY: test
test:
	go test `glide novendor`

.PHONY: image
image:
	./build.sh

.PHONY: run-image
run-image:
# Run a pre-built image locally. When the agent terminates or you ctrl-c the container is removed.
# Setup: cp -r etc local-etc and make any changes necessary to agent.yaml.
	docker run -it --rm \
		--name $(RUN_CONTAINER) \
		-e SIGNALFX_API_TOKEN=$(SIGNALFX_API_TOKEN) \
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