#!/bin/bash

set -eo pipefail

[ -f ~/.skip ] && (echo "Found ~/.skip, skipping cri-o installation" && exit 0)

CRIO_VERSION="${CRIO_VERSION:-1.15}"

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
sudo add-apt-repository -y ppa:projectatomic/ppa
sudo apt-get update
sudo apt-get install -y cri-o-${CRIO_VERSION}
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
