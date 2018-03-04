FROM ubuntu:16.04

RUN apt update &&\
    apt install -y wget vim

RUN wget -O /tmp/chefdk.deb https://packages.chef.io/files/stable/chefdk/2.4.17/ubuntu/16.04/chefdk_2.4.17-1_amd64.deb &&\
    dpkg -i /tmp/chefdk.deb

WORKDIR /chef-repo/cookbooks/signalfx_agent
COPY ./ ./
