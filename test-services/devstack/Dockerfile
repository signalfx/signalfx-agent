FROM ubuntu:18.04

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
    git \
    glusterfs-client \
    gnupg2 \
    ipcalc \
    iproute2 \
    iptables \
    kmod \
    mysql-server \
    nfs-common \
    netcat \
    rabbitmq-server \
    socat \
    software-properties-common \
    sudo \
    systemd

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

RUN systemctl enable mysql
RUN systemctl enable rabbitmq-server

RUN useradd -s /bin/bash -d /opt/stack -m stack && echo "stack ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/stack && \
    usermod -a -G mysql,rabbitmq stack

RUN su - stack -c 'git clone git://git.openstack.org/openstack-dev/devstack -b stable/rocky /opt/stack/devstack'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/cinder -b stable/rocky /opt/stack/cinder'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/glance -b stable/rocky /opt/stack/glance'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/keystone -b stable/rocky /opt/stack/keystone'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/neutron -b stable/rocky /opt/stack/neutron'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/nova -b stable/rocky /opt/stack/nova'
RUN su - stack -c 'git clone git://git.openstack.org/openstack/requirements -b stable/rocky /opt/stack/requirements'

RUN find /opt/stack/ -name "test-requirements.txt" -delete
RUN find /opt/stack/ -mindepth 3 -name "*requirements.txt" -delete

COPY local.conf /opt/stack/devstack/local.conf
RUN chown stack:stack /opt/stack/devstack/local.conf

COPY start-devstack.sh /usr/local/bin/start-devstack.sh
RUN chmod a+x /usr/local/bin/start-devstack.sh

COPY stop-devstack.sh /usr/local/bin/stop-devstack.sh
RUN chmod a+x /usr/local/bin/stop-devstack.sh

VOLUME /lib/modules
VOLUME /sys/fs/cgroup

ENTRYPOINT ["/lib/systemd/systemd"]
