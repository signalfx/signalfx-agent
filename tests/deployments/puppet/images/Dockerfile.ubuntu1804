FROM ubuntu:18.04

RUN apt-get update &&\
    apt-get install -yq ca-certificates procps systemd wget apt-transport-https libcap2-bin

RUN wget https://apt.puppetlabs.com/puppet-release-bionic.deb && \
    dpkg -i puppet-release-bionic.deb && \
    apt-get update && \
    apt-get install -y puppet

ENV PATH=/opt/puppetlabs/bin:$PATH

ENV container docker
RUN rm -f /lib/systemd/system/multi-user.target.wants/* \
    /etc/systemd/system/*.wants/* \
    /lib/systemd/system/local-fs.target.wants/* \
    /lib/systemd/system/sockets.target.wants/*udev* \
    /lib/systemd/system/sockets.target.wants/*initctl* \
    /lib/systemd/system/systemd-update-utmp*

RUN systemctl set-default multi-user.target
ENV init /lib/systemd/systemd

COPY deployments/puppet /etc/puppet/code/modules/signalfx_agent

VOLUME [ "/sys/fs/cgroup" ]

ENTRYPOINT ["/lib/systemd/systemd"]
