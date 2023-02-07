#!/bin/bash

set -eo pipefail

CRIO_VERSION="${CRIO_VERSION:-1.18}"

sudo modprobe overlay
sudo modprobe br_netfilter
cat > /tmp/99-kubernetes-cri.conf <<EOF
net.bridge.bridge-nf-call-iptables  = 1
net.ipv4.ip_forward                 = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
sudo mv -f /tmp/99-kubernetes-cri.conf /etc/sysctl.d/99-kubernetes-cri.conf
sudo sysctl --system

sudo apt-get update
sudo apt-get install -y software-properties-common

OS=xUbuntu_20.04
echo "deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/$OS/ /"|sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list
echo "deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable:/cri-o:/$CRIO_VERSION/$OS/ /"|sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable:cri-o:$CRIO_VERSION.list

curl -L https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable:cri-o:$CRIO_VERSION/$OS/Release.key | sudo apt-key add -
curl -L https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/$OS/Release.key | sudo apt-key add -

sudo apt update
sudo apt install cri-o cri-o-runc -y

sudo sed -i "s|conmon = .*|conmon = \"$(command -v conmon)\"|" /etc/crio/crio.conf
sudo mkdir -p /etc/containers
cat > /tmp/registries.conf <<EOF
[registries.search]
registries = ['docker.io', 'quay.io']
[registries.insecure]
registries = ['localhost:5000']
EOF
sudo mv -f /tmp/registries.conf /etc/containers/registries.conf
sudo systemctl start crio
sudo systemctl status --no-pager crio
