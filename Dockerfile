ARG GO_VERSION=1.12.1

###### Agent Build Image ########
FROM ubuntu:16.04 as agent-builder

RUN apt update &&\
    apt install -y curl wget pkg-config parallel

ARG GO_VERSION
ARG TARGET_ARCH

ENV PATH=$PATH:/usr/local/go/bin
RUN cd /tmp &&\
    wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-${TARGET_ARCH}.tar.gz &&\
	tar -C /usr/local -xf go*.tar.gz

ENV GOPATH=/go
WORKDIR /usr/src/signalfx-agent

COPY vendor/ ./vendor/
# Precompile and cache vendor compilation outputs so that building the app is
# faster.  A bunch of these fail because go get pulls in more than necessary, but
# a lot do compile
RUN cd vendor && find . -type d -not -empty | grep -v '\btest' | parallel go install -mod vendor {} 2>/dev/null || true

COPY cmd/ ./cmd/
COPY scripts/make-templates scripts/make-versions ./scripts/
COPY scripts/collectd-template-to-go ./scripts/
COPY Makefile .
COPY go.mod go.sum ./
COPY internal/ ./internal/

ARG collectd_version=""
ARG agent_version="latest"
ARG GOOS="linux"

RUN AGENT_VERSION=${agent_version} COLLECTD_VERSION=${collectd_version} make signalfx-agent &&\
    mv signalfx-agent /usr/bin/signalfx-agent

###### Collectd builder image ######
FROM ubuntu:16.04 as collectd

ARG TARGET_ARCH

ENV DEBIAN_FRONTEND noninteractive

RUN sed -i -e '/^deb-src/d' /etc/apt/sources.list &&\
    apt-get update &&\
    apt-get install -y \
      curl \
      dpkg \
      net-tools \
      openjdk-8-jdk \
      python-software-properties \
	  software-properties-common \
      wget \
      autoconf \
      automake \
      autotools-dev \
      bison \
      build-essential \
      debhelper \
      debian-archive-keyring \
      debootstrap \
      devscripts \
      dh-make \
      dpatch \
      fakeroot \
      flex \
      gcc \
      git-core \
      iptables-dev \
      libatasmart-dev \
      libcurl4-openssl-dev \
      libdbi0-dev \
      libdistro-info-perl \
      libesmtp-dev \
      libganglia1-dev \
      libgcrypt11-dev \
      libglib2.0-dev \
      libldap2-dev \
      libltdl-dev \
      libmemcached-dev \
      libmicrohttpd-dev \
      libmnl-dev \
      libmodbus-dev \
      libnotify-dev \
      libopenipmi-dev \
      liboping-dev \
      libow-dev \
      libpcap-dev \
      libperl-dev \
      libpq-dev \
      libprotobuf-c0-dev \
      librabbitmq-dev \
      librdkafka-dev \
      librrd-dev \
      libsensors4-dev \
      libsnmp-dev \
      libtool \
      libudev-dev \
      libvarnishapi-dev \
      libvirt-dev \
      libxml2-dev \
      libyajl-dev \
      lsb-release \
      pbuilder \
      pkg-config \
      po-debconf \
      protobuf-c-compiler \
      python-dev \
      python-pip \
      python-virtualenv \
      quilt

RUN wget https://dev.mysql.com/get/mysql-apt-config_0.8.12-1_all.deb && \
    dpkg -i mysql-apt-config_0.8.12-1_all.deb && \
    apt-get update && apt-get install -y libmysqlclient-dev libcurl4-gnutls-dev patchelf

ARG collectd_version=""
ARG collectd_commit=""

RUN cd /tmp &&\
    wget https://github.com/signalfx/collectd/archive/${collectd_commit}.tar.gz &&\
	tar -xvf ${collectd_commit}.tar.gz &&\
	mkdir -p /usr/src/ &&\
	mv collectd-${collectd_commit}* /usr/src/collectd

# Hack to get our custom version compiled into collectd
RUN echo "#!/bin/bash" > /usr/src/collectd/version-gen.sh &&\
    echo "printf \${collectd_version//-/.}" >> /usr/src/collectd/version-gen.sh

WORKDIR /usr/src/collectd

ARG extra_cflags="-O2"
ENV CFLAGS "-Wall -fPIC $extra_cflags"
ENV CXXFLAGS $CFLAGS

# In the bundle, the java plugin will live in /plugins/collectd and the JVM
# exists at /jvm
ENV JAVA_LDFLAGS "-Wl,-rpath -Wl,\$\$\ORIGIN/../../jvm/jre/lib/${TARGET_ARCH}/server"

# turbostat is not supported by ARM, let it be a ARG
ARG DISABLE_TURBOSTAT

RUN autoreconf -vif &&\
    ./configure \
        --prefix="/usr" \
        --localstatedir="/var" \
        --sysconfdir="/etc/collectd" \
        --enable-all-plugins \
        --disable-apple_sensors \
        --disable-aquaero \
        --disable-barometer \
        --disable-dpdkstat \
        --disable-dpdkevents \
        --disable-gps \
        --disable-grpc \
        --disable-intel_pmu \
        --disable-intel_rdt \
        --disable-lpar \
        --disable-lua \
        --disable-lvm \
        --disable-mic \
        --disable-mqtt \
        --disable-netapp \
        --disable-nut \
        --disable-oracle \
        --disable-pf \
        --disable-redis \
        --disable-routeros \
        --disable-sigrok \
        --disable-tape \
        --disable-tokyotyrant \
        ${DISABLE_TURBOSTAT} \
        --disable-write_mongodb \
        --disable-write_redis \
        --disable-write_riemann \
        --disable-xmms \
        --disable-zone \
        --without-included-ltdl \
        --without-libstatgrab \
        --disable-silent-rules \
        --disable-static

# Compile all of collectd first, including plugins
RUN make -j8 &&\
    make install

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /usr/sbin/collectd /usr/lib/collectd/
# For some reason libvarnishapi doesn't properly depend on libm, so make it
# right.
RUN patchelf --add-needed libm-2.23.so /opt/deps/libvarnishapi.so.1.0.4



###### Python Plugin Image ######
FROM ubuntu:16.04 as python-plugins

RUN apt update &&\
    apt install -y git python-pip wget curl &&\
    pip install --upgrade 'pip==18.0'

RUN pip install yq &&\
    wget -O /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 &&\
    chmod +x /usr/bin/jq

RUN apt install -y libffi-dev libssl-dev build-essential python-dev libcurl4-openssl-dev

# Mirror the same dir structure that exists in the original source
COPY scripts/get-collectd-plugins.py /opt/scripts/
COPY scripts/get-collectd-plugins-requirements.txt /opt/
COPY collectd-plugins.yaml /opt/

RUN pip install -r /opt/get-collectd-plugins-requirements.txt

RUN mkdir -p /opt/collectd-python &&\
    python /opt/scripts/get-collectd-plugins.py /opt/collectd-python

COPY python/ /opt/sfxpython/
RUN cd /opt/sfxpython && pip install .

# Delete all compiled python to save space
RUN find /usr/lib/python2.7 /usr/local/lib/python2.7/dist-packages -name "*.pyc" | xargs rm

####### Extra packages that don't make sense to pull down in any other stage ########
FROM ubuntu:16.04 as extra-packages

ARG TARGET_ARCH

RUN apt update &&\
    apt install -y \
	  host \
	  netcat.openbsd \
	  netcat \
	  iproute2 \
	  curl \
	  vim

RUN apt install -y openjdk-8-jre-headless &&\
    cp -rL /usr/lib/jvm/java-8-openjdk-${TARGET_ARCH} /opt/jvm &&\
	rm -rf /opt/jvm/docs &&\
	rm -rf /opt/jvm/man

RUN curl -Lo /opt/signalfx_types_db https://raw.githubusercontent.com/signalfx/integrations/master/collectd-java/signalfx_types_db

COPY scripts/collect-libs /opt/collect-libs

ENV useful_bins=" \
  /bin/bash \
  /bin/cat \
  /bin/cp \
  /bin/date \
  /bin/echo \
  /bin/grep \
  /bin/kill \
  /bin/ln \
  /bin/ls \
  /bin/mkdir \
  /bin/mount \
  /bin/nc \
  /bin/nc.openbsd \
  /bin/ps \
  /bin/rm \
  /bin/sh \
  /bin/ss \
  /bin/umount \
  /usr/bin/curl \
  /usr/bin/dirname \
  /usr/bin/host \
  /usr/bin/tail \
  /usr/bin/vim \
  "
RUN /opt/collect-libs /opt/deps ${useful_bins}

RUN mkdir -p /opt/bins &&\
    cp $useful_bins /opt/bins

###### Final Agent Image #######
# This build stage is meant as the final target when running the agent in a
# container environment (e.g. directly with Docker or on K8s).  The stages
# below this are special-purpose.
FROM scratch as final-image

ARG CPU_ARCH
ARG LDSO_BIN

CMD ["/bin/signalfx-agent"]

COPY --from=collectd /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=collectd /usr/sbin/collectd /bin/collectd
COPY --from=collectd /opt/deps/ /lib
COPY --from=collectd /etc/nsswitch.conf /etc/nsswitch.conf

COPY --from=extra-packages /lib/${CPU_ARCH}-linux-gnu/ld-2.23.so ${LDSO_BIN}
COPY --from=extra-packages /opt/jvm/ /jvm
COPY --from=extra-packages /opt/signalfx_types_db /plugins/collectd/java/
COPY --from=extra-packages /opt/deps/ /lib
COPY --from=extra-packages /opt/bins/ /bin

COPY --from=collectd /usr/share/collectd/postgresql_default.conf /plugins/collectd/postgresql_default.conf
COPY --from=collectd /usr/share/collectd/types.db /plugins/collectd/types.db
# All the built-in collectd plugins
COPY --from=collectd /usr/lib/collectd/*.so /plugins/collectd/
COPY --from=collectd /usr/share/collectd/java/ /plugins/collectd/java/

# Pull in non-C collectd plugins
COPY --from=python-plugins /opt/collectd-python/ /plugins/collectd
# Grab pip dependencies too
COPY --from=python-plugins /usr/lib/python2.7/ /lib/python2.7
COPY --from=python-plugins /usr/local/lib/python2.7/ /lib/python2.7
COPY --from=python-plugins /usr/bin/python /bin/python

COPY scripts/umount-hostfs-non-persistent /bin/umount-hostfs-non-persistent
COPY deployments/docker/agent.yaml /etc/signalfx/agent.yaml

RUN mkdir -p /run/collectd /var/run/ &&\
    ln -s /var/run/signalfx-agent /run &&\
    ln -s /bin/signalfx-agent /bin/agent-status

COPY --from=agent-builder /usr/bin/signalfx-agent /bin/signalfx-agent

COPY whitelist.json /lib/whitelist.json

WORKDIR /


# Workaround to utilize the global GO_VERSION argument
# since "COPY --from" doesn't support variables.
FROM golang:${GO_VERSION}-stretch as golang-ignore


####### Dev Image ########
# This is an image to facilitate development of the agent.  It installs all of
# the build tools for building collectd and the go agent, along with some other
# useful utilities.  The agent image is copied from the final-image stage to
# the /bundle dir in here and the SIGNALFX_BUNDLE_DIR is set to point to that.
FROM ubuntu:18.04 as dev-extras

ARG TARGET_ARCH

RUN apt update &&\
    apt install -y \
      curl \
      git \
      inotify-tools \
      iproute2 \
      jq \
      net-tools \
      python3-pip \
      socat \
      vim \
      wget


ENV SIGNALFX_BUNDLE_DIR=/bundle \
    TEST_SERVICES_DIR=/usr/src/signalfx-agent/test-services \
    AGENT_BIN=/usr/src/signalfx-agent/signalfx-agent \
    PYTHONPATH=/usr/src/signalfx-agent/python \
    AGENT_VERSION=latest \
    BUILD_TIME=2017-01-25T13:17:17-0500 \
    GOOS=linux \
    LC_ALL=C.UTF-8 \
    LANG=C.UTF-8

RUN pip3 install ipython ipdb

# Install helm
ARG HELM_VERSION=v2.13.0
WORKDIR /tmp
RUN wget -O helm.tar.gz https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-${TARGET_ARCH}.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz && \
    mv linux-${TARGET_ARCH}/helm /usr/local/bin/helm && \
    chmod a+x /usr/local/bin/helm

WORKDIR /usr/src/signalfx-agent
CMD ["/bin/bash"]
ENV PATH=$PATH:/usr/local/go/bin:/go/bin GOPATH=/go

COPY --from=golang-ignore /usr/local/go/ /usr/local/go

RUN curl -fsSL get.docker.com -o /tmp/get-docker.sh &&\
    sh /tmp/get-docker.sh

RUN go get -u golang.org/x/lint/golint &&\
    if [ `uname -m` != "aarch64" ]; then go get github.com/derekparker/delve/cmd/dlv; fi &&\
    go get github.com/tebeka/go2xunit

# Get integration test deps in here
COPY python/setup.py /tmp/
RUN pip3 install -e /tmp/
COPY tests/requirements.txt /tmp/
RUN pip3 install --upgrade pip==9.0.1 && pip3 install -r /tmp/requirements.txt
RUN wget -O /usr/bin/gomplate https://github.com/hairyhenderson/gomplate/releases/download/v2.4.0/gomplate_linux-${TARGET_ARCH}-slim &&\
    chmod +x /usr/bin/gomplate
RUN ln -s /usr/bin/python3 /usr/bin/python &&\
    ln -s /usr/bin/pip3 /usr/bin/pip

COPY --from=final-image /bin/signalfx-agent ./signalfx-agent

COPY --from=final-image / /bundle/
COPY ./ ./

####### Pandoc Converter ########
FROM ubuntu:16.04 as pandoc-converter

RUN apt update &&\
    apt install -y pandoc

COPY docs/signalfx-agent.1.man /tmp/signalfx-agent.1.man
# Create the man page for the agent
RUN mkdir /docs &&\
    pandoc --standalone --to man /tmp/signalfx-agent.1.man -o /docs/signalfx-agent.1


####### Debian Packager #######
FROM debian:9 as debian-packager

RUN apt update &&\
    apt install -y dh-make devscripts dh-systemd apt-utils awscli

ARG agent_version="latest"
WORKDIR /opt/signalfx-agent_${agent_version}

ENV DEBEMAIL="support+deb@signalfx.com" DEBFULLNAME="SignalFx, Inc."

COPY packaging/deb/debian/ ./debian
COPY packaging/etc/init.d/signalfx-agent.debian ./debian/signalfx-agent.init
COPY packaging/etc/systemd/signalfx-agent.service ./debian/signalfx-agent.service
COPY packaging/etc/systemd/signalfx-agent.tmpfile ./debian/signalfx-agent.tmpfile
COPY packaging/etc/logrotate.d/signalfx-agent.conf ./debian/signalfx-agent.logrotate
COPY packaging/deb/make-changelog ./make-changelog
COPY packaging/deb/add-output-to-repo ./add-output-to-repo
COPY packaging/deb/devscripts.conf /etc/devscripts.conf
COPY --from=pandoc-converter /docs/signalfx-agent.1 ./signalfx-agent.1

COPY packaging/etc/agent.yaml ./agent.yaml

COPY --from=final-image / ./signalfx-agent/
# Remove the agent config so it doesn't confuse people in the final output.
RUN rm -rf ./signalfx-agent/etc/signalfx


###### RPM Packager #######
FROM fedora:27 as rpm-packager

RUN yum install -y rpmdevtools createrepo rpm-sign awscli

WORKDIR /root/rpmbuild

COPY packaging/etc/agent.yaml ./SOURCES/agent.yaml
COPY packaging/etc/init.d/signalfx-agent.rhel ./SOURCES/signalfx-agent.init
COPY packaging/etc/systemd/ ./SOURCES/systemd/
COPY packaging/rpm/signalfx-agent.spec ./SPECS/signalfx-agent.spec
COPY packaging/rpm/add-output-to-repo ./add-output-to-repo
COPY --from=pandoc-converter /docs/signalfx-agent.1 ./SOURCES/signalfx-agent.1

COPY --from=final-image / ./SOURCES/signalfx-agent/
# Remove the agent config so it doesn't confuse people in the final output.
RUN rm -rf ./SOURCES/signalfx-agent/etc/signalfx
