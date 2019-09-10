FROM ubuntu:16.04

RUN sed -i -e 's|^deb http://.*\.ubuntu.com/ubuntu/\?|deb mirror://mirrors.ubuntu.com/mirrors.txt|' /etc/apt/sources.list
RUN apt update &&\
    apt install -y wget vim apt-transport-https

RUN wget -O /tmp/chefdk.deb https://packages.chef.io/files/stable/chefdk/3.7.23/ubuntu/16.04/chefdk_3.7.23-1_amd64.deb &&\
    dpkg -i /tmp/chefdk.deb

COPY ./ /tmp/cookbook
WORKDIR /chef-repo

RUN berks vendor -b /tmp/cookbook/Berksfile cookbooks/
