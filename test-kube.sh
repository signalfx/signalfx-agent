#!/bin/bash -ex

mkdir -p test_output
py3clean tests
pytest -n auto --junitxml=test_output/integration_tests.xml --html=test_output/integration_tests.html --self-contained-html -m kubernetes --k8s_version=all --verbose tests
