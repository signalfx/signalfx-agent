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
  --metrics-addr 127.0.0.1:9323 \
  --experimental \
  &> /var/log/docker.log 2>&1 < /dev/null &

minikube config set ShowBootstrapperDeprecationNotification false || true

K8S_VERSION=${K8S_VERSION:-"latest"}
if [ "$K8S_VERSION" = "latest" ]; then
    curl -sSl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
    chmod a+x /usr/local/bin/kubectl
    minikube start --vm-driver=none --bootstrapper=localkube
else
    curl -sSl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
    chmod a+x /usr/local/bin/kubectl
    minikube start --kubernetes-version $K8S_VERSION --vm-driver=none --bootstrapper=localkube
fi
sleep 2
minikube status

# wait for the cluster to be ready
TIMEOUT=${TIMEOUT:-"300"}
START_TIME=`date +%s`
while [ 0 ]; do
    if [ $(expr `date +%s` - $START_TIME) -gt $TIMEOUT ]; then
        if [ -f /var/lib/localkube/localkube.out ]; then
            echo
            echo "/var/lib/localkube/localkube.out:"
            cat /var/lib/localkube/localkube.out
            echo
        fi
        if [ -f /var/lib/localkube/localkube.err ]; then
            echo
            echo "/var/lib/localkube/localkube.err:"
            cat /var/lib/localkube/localkube.err
            echo
        fi
        kubectl get pods --all-namespaces
        echo "Timed out after $TIMEOUT seconds waiting for the cluster to be ready!"
        exit 1
    fi
    npods=`kubectl get pods --all-namespaces 2>/dev/null | grep 'kube-system' | wc -l`
    nrunning=`kubectl get pods --all-namespaces 2>/dev/null | grep 'kube-system' | grep 'Running' | wc -l`
    if [[ $npods -gt 1 && $npods -eq $nrunning ]]; then
        sleep 5
        break
    fi
    sleep 5
done

kubectl get pods --all-namespaces

kubectl config view --merge=true --flatten=true > /kubeconfig

exit 0
