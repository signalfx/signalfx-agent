ARG GO_VERSION=1.13.1

###### Agent Build Image ########
FROM ubuntu:16.04 as agent-builder

RUN apt update &&\
    apt install -y curl wget pkg-config parallel git

ARG GO_VERSION
ARG TARGET_ARCH

ENV PATH=$PATH:/usr/local/go/bin
RUN cd /tmp &&\
    wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-${TARGET_ARCH}.tar.gz &&\
	tar -C /usr/local -xf go*.tar.gz

ENV GOPATH=/go
WORKDIR /usr/src/signalfx-agent

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY scripts/collectd-template-to-go scripts/make-versions ./scripts/
COPY Makefile .
COPY pkg/ ./pkg/
RUN chmod 644 pkg/monitors/collectd/signalfx_types.db

ARG collectd_version=""
ARG agent_version="latest"
ARG GOOS="linux"

RUN AGENT_VERSION=${agent_version} COLLECTD_VERSION=${collectd_version} make signalfx-agent &&\
    mv signalfx-agent /usr/bin/signalfx-agent


###### Collectd builder image ######
FROM ubuntu:16.04 as collectd

ARG TARGET_ARCH
ARG PYTHON_VERSION=2.7.16

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
      libexpat1-dev \
      libffi-dev \
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
      libssl-dev \
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
      quilt \
      zlib1g-dev \
      libdbus-glib-1-dev \
      libdbus-1-dev

RUN wget https://dev.mysql.com/get/mysql-apt-config_0.8.12-1_all.deb && \
    dpkg -i mysql-apt-config_0.8.12-1_all.deb && \
    apt-get update && apt-get install -y libmysqlclient-dev libcurl4-gnutls-dev

ENV PYTHONHOME=/opt/python
RUN wget -O /tmp/Python-${PYTHON_VERSION}.tgz https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tgz &&\
    cd /tmp &&\
    tar xzf Python-${PYTHON_VERSION}.tgz && \
    cd Python-${PYTHON_VERSION} && \
    ./configure --prefix=$PYTHONHOME --enable-shared --enable-ipv6 --enable-unicode=ucs4 --with-system-ffi --with-system-expat && \
    make && make install

RUN echo "$PYTHONHOME/lib" > /etc/ld.so.conf.d/python.conf && \
    ldconfig $PYTHONHOME/lib
ENV PATH=$PYTHONHOME/bin:$PATH

RUN wget -O /tmp/get-pip.py https://bootstrap.pypa.io/get-pip.py && \
    python /tmp/get-pip.py 'pip==18.0'

# Compile patchelf statically from source
RUN cd /tmp &&\
    wget https://nixos.org/releases/patchelf/patchelf-0.10/patchelf-0.10.tar.gz &&\
    tar -xf patchelf*.tar.gz &&\
    cd patchelf-0.10 &&\
    ./configure LDFLAGS="-static" &&\
    make &&\
    make install

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

# In the bundle, the java plugin so will live in /lib/collectd and the JVM
# exists at /jre
ENV JAVA_LDFLAGS "-Wl,-rpath -Wl,\$\$\ORIGIN/../../jre/lib/${TARGET_ARCH}/server"

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
        --disable-ipmi \
        --disable-oracle \
        --disable-pf \
        --disable-redis \
        --disable-routeros \
        --disable-sigrok \
        --disable-tape \
        --disable-tokyotyrant \
        --disable-turbostat \
        --disable-write_mongodb \
        --disable-write_redis \
        --disable-write_riemann \
        --disable-xmms \
        --disable-zone \
        --without-libstatgrab \
        --disable-silent-rules \
        --disable-static \
        PYTHON_CONFIG="$PYTHONHOME/bin/python-config"

# Compile all of collectd first, including plugins
RUN make -j`nproc` &&\
    make install

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /usr/sbin/collectd /usr/lib/collectd/
# For some reason libvarnishapi doesn't properly depend on libm, so make it
# right.
RUN patchelf --add-needed libm-2.23.so /opt/deps/libvarnishapi.so.1.0.4

# Delete all compiled python to save space
RUN find $PYTHONHOME -name "*.pyc" -o -name "*.pyo" | xargs rm
# We don't support compiling extension modules so don't need this directory
RUN rm -rf $PYTHONHOME/lib/python2.7/config-*-linux-gnu


###### Python Plugin Image ######
FROM collectd as python-plugins

RUN pip install yq &&\
    wget -O /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 &&\
    chmod +x /usr/bin/jq

# Mirror the same dir structure that exists in the original source
COPY scripts/get-collectd-plugins.py /opt/scripts/
COPY scripts/get-collectd-plugins-requirements.txt /opt/
COPY collectd-plugins.yaml /opt/

RUN pip install -r /opt/get-collectd-plugins-requirements.txt

RUN pip install dbus-python

RUN mkdir -p /opt/collectd-python &&\
    python /opt/scripts/get-collectd-plugins.py /opt/collectd-python

COPY python/ /opt/sfxpython/
RUN cd /opt/sfxpython && pip install .

RUN pip list

# Delete all compiled python to save space
RUN find $PYTHONHOME -name "*.pyc" -o -name "*.pyo" | xargs rm
# We don't support compiling extension modules so don't need this directory
RUN rm -rf $PYTHONHOME/lib/python2.7/config-*-linux-gnu


######### Java monitor dependencies and monitor jar compilation
FROM ubuntu:16.04 as java

RUN apt update &&\
    apt install -y openjdk-8-jdk maven

ARG TARGET_ARCH

RUN mkdir -p /opt/root &&\
    cp -rL /usr/lib/jvm/java-8-openjdk-${TARGET_ARCH}/jre /opt/root/jre &&\
	rm -rf /opt/root/jre/man

COPY java/ /usr/src/agent-java/
RUN cd /usr/src/agent-java/runner &&\
    mvn clean install

RUN cd /usr/src/agent-java/jmx &&\
    mvn clean package


####### Extra packages that don't make sense to pull down in any other stage ########
FROM ubuntu:16.04 as extra-packages

RUN apt update &&\
    apt install -y \
	  curl \
	  host \
	  iproute2 \
	  netcat \
	  netcat.openbsd \
	  realpath \
	  vim

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
  /bin/ps \
  /bin/rm \
  /bin/sh \
  /bin/ss \
  /bin/tar \
  /bin/umount \
  /usr/bin/curl \
  /usr/bin/dirname \
  /usr/bin/find \
  /usr/bin/host \
  /usr/bin/realpath \
  /usr/bin/tail \
  /usr/bin/vim \
  "
RUN mkdir -p /opt/root/lib &&\
    /opt/collect-libs /opt/root/lib ${useful_bins}

RUN mkdir -p /opt/root/bin &&\
    cp $useful_bins /opt/root/bin

# Gather all our bins/libs and set rpath on the properly.  Interpreter has to
# be set at runtime (or in the final docker stage for docker runs).
COPY --from=collectd /usr/local/bin/patchelf /usr/bin/

# Gather Python dependencies
COPY --from=python-plugins /opt/python/lib/python2.7 /opt/root/lib/python2.7
COPY --from=python-plugins /opt/python/lib/libpython2.7.so.1.0 /opt/root/lib
COPY --from=python-plugins /opt/python/bin/python /opt/root/bin/python

# Gather compiled collectd plugin libraries
COPY --from=collectd /usr/sbin/collectd /opt/root/bin/collectd
COPY --from=collectd /opt/deps/ /opt/root/lib/
COPY --from=collectd /usr/lib/collectd/*.so /opt/root/lib/collectd/

COPY --from=java /opt/root/jre/ /opt/root/jre/

COPY scripts/patch-rpath /usr/bin/
RUN patch-rpath /opt/root


###### Final Agent Image #######
# This build stage is meant as the final target when running the agent in a
# container environment (e.g. directly with Docker or on K8s).  The stages
# below this are special-purpose.
FROM scratch as final-image

CMD ["/bin/signalfx-agent"]

COPY --from=collectd /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_FILE /etc/ssl/certs/ca-certificates.crt

COPY --from=collectd /etc/nsswitch.conf /etc/nsswitch.conf
COPY --from=collectd /usr/local/bin/patchelf /bin/

# Pull in the Linux dynamic link loader at a fixed path across all
# architectures.  Binaries will later be patched to use this interpreter
# natively.
COPY --from=extra-packages /lib/*-linux-gnu/ld-2.23.so /bin/ld-linux.so

# Java dependencies
COPY --from=extra-packages /opt/root/jre/ /jre
COPY --from=java /usr/src/agent-java/jmx/target/agent-jmx-monitor-1.0-SNAPSHOT.jar /lib/jmx-monitor.jar

COPY --from=extra-packages /opt/root/lib/ /lib/
COPY --from=extra-packages /opt/root/bin/ /bin/

# Some extra non-binary collectd resources
COPY --from=collectd /usr/share/collectd/postgresql_default.conf /postgresql_default.conf
COPY --from=collectd /usr/share/collectd/types.db /types.db
COPY --from=collectd /usr/share/collectd/java/ /collectd-java/
COPY --from=agent-builder /usr/src/signalfx-agent/pkg/monitors/collectd/signalfx_types.db /collectd-java/signalfx_types_db

# Pull in Python collectd plugin scripts
COPY --from=python-plugins /opt/collectd-python/ /collectd-python/

COPY scripts/umount-hostfs-non-persistent /bin/umount-hostfs-non-persistent
COPY deployments/docker/agent.yaml /etc/signalfx/agent.yaml
COPY scripts/patch-interpreter /bin/
RUN ["/bin/ld-linux.so", "/bin/sh", "/bin/patch-interpreter", "/"]

RUN mkdir -p /run/collectd /var/run/ &&\
    ln -s /var/run/signalfx-agent /run &&\
    ln -s /bin/signalfx-agent /bin/agent-status

COPY --from=agent-builder /usr/bin/signalfx-agent /bin/signalfx-agent

COPY whitelist.json /lib/whitelist.json

WORKDIR /


####### Dev Image ########
# This is an image to facilitate development of the agent.  It installs all of
# the build tools for building collectd and the go agent, along with some other
# useful utilities.  The agent image is copied from the final-image stage to
# the /bundle dir in here and the SIGNALFX_BUNDLE_DIR is set to point to that.
FROM ubuntu:18.04 as dev-extras

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
      sudo \
      vim \
      wget

ENV PATH=$PATH:/usr/local/go/bin:/go/bin GOPATH=/go
ENV SIGNALFX_BUNDLE_DIR=/bundle \
    TEST_SERVICES_DIR=/usr/src/signalfx-agent/test-services \
    AGENT_BIN=/usr/src/signalfx-agent/signalfx-agent \
    PYTHONPATH=/usr/src/signalfx-agent/python \
    AGENT_VERSION=latest \
    BUILD_TIME=2017-01-25T13:17:17-0500 \
    GOOS=linux \
    LC_ALL=C.UTF-8 \
    LANG=C.UTF-8

RUN curl -fsSL get.docker.com -o /tmp/get-docker.sh &&\
    sh /tmp/get-docker.sh

# Get integration test deps in here
RUN pip3 install ipython ipdb
COPY tests/requirements.txt /tmp/
RUN pip3 install --upgrade pip==9.0.1 && pip3 install -r /tmp/requirements.txt
RUN ln -s /usr/bin/python3 /usr/bin/python &&\
    ln -s /usr/bin/pip3 /usr/bin/pip

ARG TARGET_ARCH

RUN wget -O /usr/bin/gomplate https://github.com/hairyhenderson/gomplate/releases/download/v3.4.0/gomplate_linux-${TARGET_ARCH} &&\
    chmod +x /usr/bin/gomplate

# Install helm
ARG HELM_VERSION=v3.0.0
WORKDIR /tmp
RUN wget -O helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGET_ARCH}.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz && \
    mv linux-${TARGET_ARCH}/helm /usr/local/bin/helm && \
    chmod a+x /usr/local/bin/helm

# Install kubectl
ARG KUBECTL_VERSION=v1.14.1
RUN cd /tmp &&\
    curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${TARGET_ARCH}/kubectl &&\
    chmod +x ./kubectl &&\
    mv ./kubectl /usr/bin/kubectl

WORKDIR /usr/src/signalfx-agent

COPY --from=final-image /bin/signalfx-agent ./signalfx-agent
COPY --from=final-image / /bundle/
RUN /bundle/bin/patch-interpreter /bundle

COPY --from=agent-builder /usr/local/go /usr/local/go
COPY --from=agent-builder /go $GOPATH

RUN go get -u golang.org/x/lint/golint &&\
    if [ `uname -m` != "aarch64" ]; then go get github.com/derekparker/delve/cmd/dlv; fi &&\
    go get github.com/tebeka/go2xunit &&\
    curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.20.0

COPY ./ ./

CMD ["/bin/bash"]


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
    apt install -y dh-make devscripts dh-systemd apt-utils

ARG agent_version="latest"
WORKDIR /opt/signalfx-agent_${agent_version}

ENV DEBEMAIL="support+deb@signalfx.com" DEBFULLNAME="SignalFx, Inc."

COPY packaging/deb/debian/ ./debian
COPY packaging/etc/init.d/signalfx-agent.debian ./debian/signalfx-agent.init
COPY packaging/etc/systemd/signalfx-agent.service ./debian/signalfx-agent.service
COPY packaging/etc/systemd/signalfx-agent.tmpfile ./debian/signalfx-agent.tmpfile
COPY packaging/etc/logrotate.d/signalfx-agent.conf ./debian/signalfx-agent.logrotate
COPY packaging/deb/make-changelog ./make-changelog
COPY packaging/deb/devscripts.conf /etc/devscripts.conf
COPY --from=pandoc-converter /docs/signalfx-agent.1 ./signalfx-agent.1

COPY packaging/etc/agent.yaml ./agent.yaml

COPY --from=final-image / /usr/lib/signalfx-agent/
# Remove the agent config so it doesn't confuse people in the final output.
RUN rm -rf /usr/lib/signalfx-agent/etc/signalfx

# Remove agent-status symlink; will be recreated in /usr/bin during packaging.
RUN rm -f /usr/lib/signalfx-agent/bin/agent-status

RUN /usr/lib/signalfx-agent/bin/patch-interpreter /usr/lib/signalfx-agent
RUN mv /usr/lib/signalfx-agent ./signalfx-agent

###### RPM Packager #######
FROM fedora:27 as rpm-packager

RUN yum install -y rpmdevtools

WORKDIR /root/rpmbuild

COPY packaging/etc/agent.yaml ./SOURCES/agent.yaml
COPY packaging/etc/init.d/signalfx-agent.rhel ./SOURCES/signalfx-agent.init
COPY packaging/etc/systemd/ ./SOURCES/systemd/
COPY packaging/rpm/signalfx-agent.spec ./SPECS/signalfx-agent.spec
COPY --from=pandoc-converter /docs/signalfx-agent.1 ./SOURCES/signalfx-agent.1

COPY --from=final-image / /usr/lib/signalfx-agent/
# Remove the agent config so it doesn't confuse people in the final output.
RUN rm -rf /usr/lib/signalfx-agent/etc/signalfx

# Remove agent-status symlink; will be recreated in /usr/bin during packaging.
RUN rm -f /usr/lib/signalfx-agent/bin/agent-status

RUN /usr/lib/signalfx-agent/bin/patch-interpreter /usr/lib/signalfx-agent/
RUN mv /usr/lib/signalfx-agent/ ./SOURCES/signalfx-agent
