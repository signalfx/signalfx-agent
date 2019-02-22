
.PHONY: dev-image
dev-image:
	docker build -t signalfx-agent-chef-dev .

.PHONY: run-dev-image
run-dev-image:
	docker exec -it signalfx-agent-chef-dev bash 2>/dev/null || \
	docker run \
		--rm \
		--name signalfx-agent-chef-dev \
		-it \
		-v $(CURDIR):/chef-repo/cookbooks/signalfx_agent \
		-v $(HOME)/.chef:/root/.chef \
		signalfx-agent-chef-dev \
		bash

.PHONY: release
release:
	./release

.PHONY: run-tests
run-tests:
	./test/run_tests.sh
