#!/bin/bash -e
# Portions Copyright 2016 The Kubernetes Authors All rights reserved.
# Portions Copyright 2018 AspenMesh
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Based on:
# https://github.com/kubernetes/minikube/tree/master/deploy/docker/localkube-dind

if [ -f /kubeconfig ]; then
    rm -f /kubeconfig
fi

mount --make-shared /
dockerd \
  --host=unix:///var/run/docker.sock \
  --host=tcp://0.0.0.0:2375 \
  --insecure-registry localhost:5000 \
  &> /var/log/docker.log 2>&1 < /dev/null &

minikube config set ShowBootstrapperDeprecationNotification false
if [ -n "$K8S_VERSION" ]; then
    minikube start --kubernetes-version $K8S_VERSION --vm-driver=none --bootstrapper=localkube
else
    minikube start --vm-driver=none --bootstrapper=localkube
fi
sleep 2
minikube status

set +e
# wait for the cluster to be ready
if [ -z "$TIMEOUT" ]; then
    TIMEOUT=300
fi
(( TIMEOUT = $TIMEOUT / 2 ))
for ((i=0; i<=${TIMEOUT}; i++)); do
    npods=`kubectl get pods --all-namespaces 2>/dev/null | grep 'kube-system' | wc -l`
    nrunning=`kubectl get pods --all-namespaces 2>/dev/null | grep 'kube-system' | grep 'Running' | wc -l`
    if [[ $npods -gt 1 && $npods -eq $nrunning ]]; then
        sleep 5
        break
    fi
    sleep 2
done
set -e

kubectl get pods --all-namespaces || exit 1

kubectl create secret generic signalfx-agent --from-literal=access-token=testing123

kubectl config view --merge=true --flatten=true > /kubeconfig

exit 0

