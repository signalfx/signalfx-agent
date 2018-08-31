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

docker version >/dev/null 2>&1 || /usr/local/bin/start-docker.sh

if [ -f /kubeconfig ]; then
    rm -f /kubeconfig
fi

K8S_VERSION=${K8S_VERSION:-"latest"}
BOOTSTRAPPER=${BOOTSTRAPPER:-"kubeadm"}
MINIKUBE_OPTIONS="--vm-driver=none --bootstrapper=${BOOTSTRAPPER}"
if [ "$K8S_VERSION" = "latest" ]; then
    curl -sSl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
else
    curl -sSl -o /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
    MINIKUBE_OPTIONS="$MINIKUBE_OPTIONS --kubernetes-version $K8S_VERSION"
fi
chmod a+x /usr/local/bin/kubectl
minikube config set ShowBootstrapperDeprecationNotification false || true
minikube config set WantNoneDriverWarning false || true

if [ "$BOOTSTRAPPER" = "kubeadm" ]; then
    # Initialize minikube but expect kubeadm to fail due to the
    # FileContent--proc-sys-net-bridge-bridge-nf-call-iptables preflight error.
    minikube start $MINIKUBE_OPTIONS --extra-config=kubelet.authorization-mode=AlwaysAllow --extra-config=kubelet.anonymous-auth=true --extra-config=kubelet.cadvisor-port=4194 || true
    # Run "kubeadm init" again but ignore the FileContent--proc-sys-net-bridge-bridge-nf-call-iptables preflight error.
    kubeadm init --config /var/lib/kubeadm.yaml --ignore-preflight-errors=DirAvailable--etc-kubernetes-manifests --ignore-preflight-errors=DirAvailable--data-minikube --ignore-preflight-errors=Port-10250 --ignore-preflight-errors=FileAvailable--etc-kubernetes-manifests-kube-scheduler.yaml --ignore-preflight-errors=FileAvailable--etc-kubernetes-manifests-kube-apiserver.yaml --ignore-preflight-errors=FileAvailable--etc-kubernetes-manifests-kube-controller-manager.yaml --ignore-preflight-errors=FileAvailable--etc-kubernetes-manifests-etcd.yaml --ignore-preflight-errors=Swap --ignore-preflight-errors=CRI --ignore-preflight-errors=FileContent--proc-sys-net-bridge-bridge-nf-call-iptables
elif [ "$BOOTSTRAPPER" = "localkube" ]; then
    mv /usr/local/bin/fake-systemctl.sh /usr/local/bin/systemctl
    chmod a+x /usr/local/bin/systemctl
    export PATH=/usr/local/bin:$PATH
    minikube start $MINIKUBE_OPTIONS
else
    echo "Unsupported bootstrapper \"${BOOTSTRAPPER}\"!"
    exit 1
fi

minikube addons disable dashboard || true
sleep 2
minikube status || true
kubectl version || true

# wait for the cluster to be ready
TIMEOUT=${TIMEOUT:-"300"}
START_TIME=`date +%s`
while [ 0 ]; do
    if [ $(expr `date +%s` - $START_TIME) -gt $TIMEOUT ]; then
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
