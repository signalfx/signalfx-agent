FROM centos:6

RUN rpm -Uvh https://yum.puppet.com/puppet-release-el-6.noarch.rpm
RUN yum install -y upstart procps udev initscripts puppet-agent

COPY tests/packaging/images/init-fake.conf /etc/init/fake-container-events.conf
COPY deployments/puppet /etc/puppetlabs/code/modules/signalfx_agent

ENV PATH=/opt/puppetlabs/bin:$PATH

CMD ["/sbin/init", "-v"]
