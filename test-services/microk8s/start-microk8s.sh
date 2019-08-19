#!/bin/bash -e

if [[ "$CIRCLECI" != "true" && "$container" = "docker" ]]; then
    exec &> >(tee -a /var/log/start-microk8s.log)
fi

K8S_VERSION=${K8S_VERSION:-"latest"}
CHANNEL=${CHANNEL:-"stable"}
KUBECONFIG_PATH=${KUBECONFIG_PATH:-"/kubeconfig"}
TIMEOUT=${TIMEOUT:-"120"}

function start_registry() {
    docker run \
        -e "REGISTRY_HTTP_ADDR=0.0.0.0:5000" \
        -d \
        -p 5000:5000 \
        --name registry \
        registry:2.7
}

function cluster_is_ready() {
    echo "Waiting for the cluster to be ready ..."
    local start_time=`date +%s`
    while [ $(expr `date +%s` - $start_time) -lt $TIMEOUT ]; do
        if kubectl get all --all-namespaces && kubectl describe serviceaccount default | grep 'default-token'; then
            return 0
        fi
        sleep 5
    done
    return 1
}

if [[ "$CIRCLECI" != "true" && "$container" = "docker" ]]; then
    mount --make-rshared /
    /usr/local/bin/start-docker.sh
fi

sudo systemctl enable snapd
sudo systemctl start snapd
sleep 5

if [ "$K8S_VERSION" = "latest" ]; then
    sudo snap install microk8s --classic --${CHANNEL}
else
    sudo snap install microk8s --classic --channel=${K8S_VERSION}/${CHANNEL}
fi
sudo snap alias microk8s.kubectl kubectl

cluster_is_ready

start_registry

kubectl version
if [[ "$CIRCLECI" != "true" && "$container" = "docker" ]]; then
    kubectl config view --merge=true --flatten=true > "$KUBECONFIG_PATH"
fi
exit 0
