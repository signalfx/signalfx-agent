FROM centos:7

RUN yum install -y wget

RUN wget -O /tmp/chefdk.rpm https://packages.chef.io/files/stable/chefdk/3.7.23/el/7/chefdk-3.7.23-1.el7.x86_64.rpm &&\
    rpm -i /tmp/chefdk.rpm

COPY ./ /tmp/cookbook
WORKDIR /chef-repo

RUN berks vendor -b /tmp/cookbook/Berksfile cookbooks/
