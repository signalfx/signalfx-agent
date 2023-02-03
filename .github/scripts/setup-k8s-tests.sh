#!/bin/bash

set -eo pipefail

K8S_SFX_AGENT="${K8S_SFX_AGENT:-quay.io/signalfx/signalfx-agent-dev:latest}"

docker run -d -e "REGISTRY_HTTP_ADDR=0.0.0.0:5000" -p 5000:5000 registry:2.7
docker run --rm --net=host jwilder/dockerize:0.6.1 -wait tcp://localhost:5000 -timeout 10s
docker tag $K8S_SFX_AGENT localhost:5000/signalfx-agent:latest
docker push localhost:5000/signalfx-agent:latest
