FROM centos:6

RUN yum install -y upstart initscripts

WORKDIR /opt/cookbooks
RUN curl -Lo windows.tar.gz https://supermarket.chef.io/cookbooks/windows/versions/4.3.4/download && \
    tar -xzf windows.tar.gz

RUN yum install -y https://packages.chef.io/files/stable/chef/12.8.1/el/6/chef-12.8.1-1.el6.x86_64.rpm

COPY tests/packaging/images/socat /bin/socat

# Insert our fake certs to the system bundle so they are trusted
COPY tests/packaging/images/certs/*.signalfx.com.* /
RUN cat /*.cert >> /etc/pki/tls/certs/ca-bundle.crt

COPY tests/packaging/images/init-fake.conf /etc/init/fake-container-events.conf

COPY deployments/chef /opt/cookbooks/signalfx_agent

WORKDIR /opt

CMD ["/sbin/init", "-v"]
