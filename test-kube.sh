#!/bin/bash -ex

docker run -d -p 5000:5000 --restart always --name registry registry 2>/dev/null || true
docker tag quay.io/signalfx/signalfx-agent-dev:stage-final-image localhost:5000/signalfx-agent-dev
docker push localhost:5000/signalfx-agent-dev

mkdir -p test_output
py3clean tests
pytest -n auto --junitxml=test_output/integration_tests.xml --html=test_output/integration_tests.html --self-contained-html -m kubernetes --k8s_version=all --verbose tests
