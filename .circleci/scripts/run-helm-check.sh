#!/bin/bash

set -eo pipefail

if [ "$CIRCLE_BRANCH" != "master" ]; then
  if ! scripts/changes-include-dir deployments/k8s ${BASH_SOURCE[0]}; then
      echo "No changes in deployments/k8s, skipping check."
      exit 0
  fi
fi

bash -ec "./deployments/k8s/generate-from-helm && git diff --exit-code" || \
    (echo 'Helm charts and generated sample K8s resources are out of sync.  Please run "./deployments/k8s/generate-from-helm" in the dev-image and commit the changes.' && exit 1)

helm lint ./deployments/k8s/helm/signalfx-agent || \
    (echo 'Helm lint issues found. Please run "helm lint ./deployments/k8s/helm/signalfx-agent" in the dev-image, resolve the issues, and commit the changes' && exit 1)
