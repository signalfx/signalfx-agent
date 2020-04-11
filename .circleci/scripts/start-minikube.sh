#!/bin/bash

set -euo pipefail

export MINIKUBE_IN_STYLE="false"
export MINIKUBE_WANTUPDATENOTIFICATION="false"
export MINIKUBE_WANTREPORTERRORPROMPT="false"
export CHANGE_MINIKUBE_NONE_USER="true"
export K8S_VERSION=${K8S_VERSION:-}
export OPTIONS=${OPTIONS:-}

[ -f ~/.skip ] && exit 0

sudo apt-get update
sudo apt-get -y install conntrack

# enable kubelet port 10255 for cadvisor and stats
OPTIONS="$OPTIONS --extra-config=kubelet.read-only-port=10255"

if [[ "$K8S_VERSION" =~ ^v1\.18\. ]]; then
    # enable kubelet cadvisor for K8S_VERSION v1.18
    OPTIONS="$OPTIONS --extra-config=kubelet.enable-cadvisor-json-endpoints=true"
fi

if [ -f test-services/minikube/config.json ]; then
    mkdir -p $HOME/.minikube/config
    cp test-services/minikube/config.json $HOME/.minikube/config/config.json
fi

if [ -n "$K8S_VERSION" ]; then
    sudo -E minikube start --vm-driver=none --wait=true --kubernetes-version=$K8S_VERSION $OPTIONS
else
    sudo -E minikube start --vm-driver=none --wait=true $OPTIONS
fi
