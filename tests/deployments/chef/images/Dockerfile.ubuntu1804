FROM ubuntu:18.04

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update &&\
    apt-get install -yq ca-certificates procps systemd apt-transport-https libcap2-bin curl

WORKDIR /opt/cookbooks
RUN curl -Lo windows.tar.gz https://supermarket.chef.io/cookbooks/windows/versions/4.3.4/download && \
    tar -xzf windows.tar.gz

RUN curl -Lo /tmp/chef.deb https://packages.chef.io/files/stable/chef/14.1.1/ubuntu/18.04/chef_14.1.1-1_amd64.deb && \
    dpkg -i /tmp/chef.deb

COPY tests/packaging/images/socat /bin/socat

# Insert our fake certs to the system bundle so they are trusted
COPY tests/packaging/images/certs/*.signalfx.com.* /
RUN cat /*.cert >> /etc/ssl/certs/ca-certificates.crt

ENV container docker
RUN rm -f /lib/systemd/system/multi-user.target.wants/* \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/systemd-update-utmp*

RUN systemctl set-default multi-user.target
ENV init /lib/systemd/systemd

COPY deployments/chef /opt/cookbooks/signalfx_agent

WORKDIR /opt

VOLUME [ "/sys/fs/cgroup" ]

ENTRYPOINT ["/lib/systemd/systemd"]
