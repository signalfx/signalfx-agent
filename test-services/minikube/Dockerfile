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

FROM ubuntu:16.04
# Install minikube dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN sed -i 's|^deb http://.*\.\?archive\.ubuntu.com|deb http://us.archive.ubuntu.com|' /etc/apt/sources.list
RUN apt-get update -y && \
    apt-get -yy -q --no-install-recommends install \
    apt-transport-https \
    aufs-tools \
    bridge-utils \
    ca-certificates \
    cifs-utils \
    conntrack \
    curl \
    ebtables \
    ethtool \
    glusterfs-client \
    gnupg2 \
    ipcalc \
    iproute2 \
    iptables \
    kmod \
    nfs-common \
    netcat \
    socat \
    software-properties-common \
    sudo \
    systemd

# Install docker
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
RUN add-apt-repository \
    "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
    $(lsb_release -cs) \
    stable"
RUN apt-get update && \
    apt-get -yy -q install docker-ce=17.12.1~ce-0~ubuntu
VOLUME /var/lib/docker
EXPOSE 2375

ARG CRI_VERSION=v1.11.1

# Install crictl for kubeadm
RUN curl -sSL -O https://github.com/kubernetes-incubator/cri-tools/releases/download/${CRI_VERSION}/crictl-${CRI_VERSION}-linux-amd64.tar.gz && \
    tar zxvf crictl-${CRI_VERSION}-linux-amd64.tar.gz -C /usr/local/bin && \
    chmod a+x /usr/local/bin/crictl && \
    rm -f crictl-${CRI_VERSION}-linux-amd64.tar.gz

ARG HELM_VERSION=v2.13.0

# Install helm
WORKDIR /tmp
RUN curl -sSl -o helm.tar.gz https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm && \
    chmod a+x /usr/local/bin/helm

ARG MINIKUBE_VERSION=latest

# Install minikube
RUN curl -sSl -o /usr/local/bin/minikube https://storage.googleapis.com/minikube/releases/${MINIKUBE_VERSION}/minikube-linux-amd64
RUN chmod a+x /usr/local/bin/minikube
ENV MINIKUBE_WANTUPDATENOTIFICATION=false \
    MINIKUBE_WANTREPORTERRORPROMPT=false \
    CHANGE_MINIKUBE_NONE_USER=true
EXPOSE 8443

ENV container docker
RUN rm -f /lib/systemd/system/multi-user.target.wants/* \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/systemd-update-utmp*

RUN systemctl set-default multi-user.target
ENV init /lib/systemd/systemd

RUN apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy local start scripts
COPY start-docker.sh /usr/local/bin/start-docker.sh
RUN chmod a+x /usr/local/bin/start-docker.sh
COPY start-minikube.sh /usr/local/bin/start-minikube.sh
RUN chmod a+x /usr/local/bin/start-minikube.sh

COPY config.json /root/.minikube/config/config.json

WORKDIR /

ENTRYPOINT ["/lib/systemd/systemd"]
