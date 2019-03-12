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

exec &> >(tee -a /var/log/start-minikube.log)

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"/kubeconfig"}
TIMEOUT=${TIMEOUT:-"300"}
K8S_VERSION=${K8S_VERSION:-"latest"}
if [ "$K8S_VERSION" = "latest" ]; then
    K8S_VERSION=`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`
fi
BOOTSTRAPPER="kubeadm"
MAJOR_VERSION=`echo ${K8S_VERSION#v} | cut -d. -f1`
MINOR_VERSION=`echo $K8S_VERSION | cut -d. -f2`
if [[ "$MAJOR_VERSION" =~ ^[0-9]+$ ]] && [[ "$MINOR_VERSION" =~ ^[0-9]+$ ]]; then
    if [[ $MAJOR_VERSION -le 1 && $MINOR_VERSION -lt 11 ]]; then
        BOOTSTRAPPER="localkube"
    fi
else
    echo "Unknown K8s version '$K8S_VERSION'!"
    exit 1
fi

KUBECTL_VERSION=$K8S_VERSION
KUBECTL_URL="https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl"

MINIKUBE_OPTIONS="--vm-driver=none --bootstrapper=${BOOTSTRAPPER} --kubernetes-version=${K8S_VERSION}"

KUBEADM_OPTIONS="--feature-gates=CoreDNS=true \
    --extra-config=kubelet.authorization-mode=AlwaysAllow \
    --extra-config=kubelet.anonymous-auth=true"
if [[ "$K8S_VERSION" =~ ^v1\.11\. ]]; then
    KUBEADM_OPTIONS="$KUBEADM_OPTIONS --extra-config=kubelet.cadvisor-port=4194"
fi

if [ "$BOOTSTRAPPER" = "kubeadm" ]; then
    MINIKUBE_OPTIONS="$MINIKUBE_OPTIONS $KUBEADM_OPTIONS"
fi

function download_kubectl() {
    [ -f /usr/local/bin/kubectl ] && rm -f /usr/local/bin/kubectl
    curl -sSl -o /usr/local/bin/kubectl $KUBECTL_URL
    chmod a+x /usr/local/bin/kubectl
}

function start_minikube() {
    if minikube delete >/dev/null 2>&1; then
        echo "Deleted minikube cluster"
    fi
    if [ "$BOOTSTRAPPER" = "kubeadm" ]; then
        # Initialize minikube but expect "kubeadm init" to fail due to preflight errors.
        if ! minikube start $MINIKUBE_OPTIONS; then
            # Run "kubeadm init" again but ignore preflight errors.
            kubeadm init --config /var/lib/kubeadm.yaml --ignore-preflight-errors=all
        fi
    elif [ "$BOOTSTRAPPER" = "localkube" ]; then
        minikube start $MINIKUBE_OPTIONS --extra-config=apiserver.Authorization.Mode=RBAC
    else
        echo "Unsupported bootstrapper \"${BOOTSTRAPPER}\"!"
        exit 1
    fi
}

function cluster_is_ready() {
    echo "Waiting for the cluster to be ready ..."
    local start_time=`date +%s`
    while [ $(expr `date +%s` - $start_time) -lt $TIMEOUT ]; do
        if kubectl get all --all-namespaces; then
            return 0
        fi
        sleep 5
    done
    return 1
}

function print_logs() {
    if [ "$BOOTSTRAPPER" = "localkube" ]; then
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
    else
        echo
        echo "minikube logs:"
        minikube logs
        echo
    fi
}

mount --make-rshared /
/usr/local/bin/start-docker.sh
download_kubectl
start_minikube
if ! cluster_is_ready; then
    print_logs
    echo "Timed out after $TIMEOUT seconds waiting for the cluster to be ready!"
    exit 1
fi
kubectl version
kubectl config view --merge=true --flatten=true > "$KUBECONFIG_PATH"
exit 0
