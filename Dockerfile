###### Collectd builder image ######
FROM ubuntu:16.04 as collectd

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
      libmysqlclient-dev \
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
      pbuilder \
      pkg-config \
      po-debconf \
      protobuf-c-compiler \
      python-dev \
      python-pip \
      python-virtualenv \
      quilt

RUN apt install -y libcurl4-gnutls-dev

COPY VERSIONS /tmp
# TODO: once neoagent-changes branch in collectd gets merged, change "collectd_file_base"
# below to "$(./VERSIONS collectd_version)" and remove the former build arg.
RUN cd /tmp &&\
    wget https://github.com/signalfx/collectd/archive/`./VERSIONS collectd_commit`.tar.gz &&\
	tar -xvf `./VERSIONS collectd_commit`.tar.gz &&\
	mkdir -p /usr/src/ &&\
	mv collectd-`./VERSIONS collectd_commit`* /usr/src/collectd

# Hack to get our custom version compiled into collectd
RUN echo "#!/bin/bash" > /usr/src/collectd/version-gen.sh &&\
    echo "collectd_version=$(/tmp/VERSIONS collectd_version)" >> /usr/src/collectd/version-gen.sh &&\
    echo "printf \${collectd_version//-/.}" >> /usr/src/collectd/version-gen.sh

WORKDIR /usr/src/collectd

ARG extra_cflags="-O2"
ENV CFLAGS "-Wall -fPIC -DSIGNALFX_EIM=0 $extra_cflags"
ENV CXXFLAGS $CFLAGS

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
	  --disable-gps \
	  --disable-grpc \
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
RUN make -j6 &&\
    make install

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /usr/sbin/collectd /usr/lib/collectd/

RUN rm -rf /usr/lib/jvm/java-8-openjdk-amd64/man /usr/lib/jvm/java-8-openjdk-amd64/docs /usr/lib/jvm/java-8-openjdk-amd64/include

###### Golang Dependencies Image ######
FROM golang:1.9.2-stretch as godeps

RUN wget -O /usr/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 &&\
    chmod +x /usr/bin/dep

WORKDIR /go/src/github.com/signalfx/neo-agent
COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

# Precompile and cache vendor objects so that building the app is faster
# A bunch of these fail because dep pulls in more than necessary, but a lot do compile
RUN cd vendor && for p in $(find . -type d -not -empty | grep -v '\btest'); do go install $p 2>/dev/null; done || true


###### Neoagent Build Image ########
FROM ubuntu:16.04 as agent-builder

# Cgo requires dep libraries present
RUN apt update &&\
    apt install -y wget pkg-config

ENV GO_VERSION=1.9.2 PATH=$PATH:/usr/local/go/bin
RUN cd /tmp &&\
    wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz &&\
	tar -C /usr/local -xf go*.tar.gz

COPY --from=godeps /go/src/github.com/signalfx/neo-agent/vendor /go/src/github.com/signalfx/neo-agent/vendor
COPY --from=godeps /go/pkg /go/pkg
COPY --from=collectd /usr/src/collectd/ /usr/src/collectd
# The agent source files are tarred up because otherwise we would have to have
# a separate ADD/COPY layer for every top-level package dir.
ADD scripts/go_packages.tar /go/src/github.com/signalfx/neo-agent/

ENV GOPATH=/go
WORKDIR /go/src/github.com/signalfx/neo-agent
COPY VERSIONS .

RUN make signalfx-agent &&\
	cp signalfx-agent /usr/bin/signalfx-agent

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /usr/bin/signalfx-agent


###### Python Plugin Image ######
FROM ubuntu:16.04 as python-plugins

RUN apt update &&\
    apt install -y git python-pip wget
RUN pip install yq &&\
    wget -O /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 &&\
    chmod +x /usr/bin/jq

# Mirror the same dir structure that exists in the original source
COPY scripts/get-collectd-plugins.sh /opt/scripts/
COPY collectd-plugins.yaml /opt/

RUN mkdir -p /usr/share/collectd/java \
    && echo "jmx_memory      value:GAUGE:0:U" > /usr/share/collectd/java/signalfx_types_db

RUN bash /opt/scripts/get-collectd-plugins.sh

RUN apt install -y libffi-dev libssl-dev build-essential python-dev libcurl4-openssl-dev

#COPY scripts/install-dd-plugin-deps.sh /opt/

#RUN mkdir -p /opt/dd &&\
    #cd /opt/dd &&\
    #git clone --depth 1 --single-branch https://github.com/DataDog/dd-agent.git &&\
	#git clone --depth 1 --single-branch https://github.com/DataDog/integrations-core.git

#RUN bash /opt/install-dd-plugin-deps.sh

COPY neopy/requirements.txt /tmp/requirements.txt
RUN pip install -r /tmp/requirements.txt

# Delete all compiled python to save space
RUN find /usr/lib/python2.7 /usr/local/lib/python2.7/dist-packages -name "*.pyc" | xargs rm

####### Extra packages to make things easier to work with ########
FROM ubuntu:16.04 as extra-packages

RUN apt update &&\
    apt install -y \
	  netcat.openbsd \
	  curl \
	  vim

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /bin /usr/bin/vim /usr/bin/curl /usr/bin/du


###### Final Agent Image #######
FROM scratch as final-image

# Pull in non-C collectd plugins
COPY --from=python-plugins /usr/share/collectd/ /usr/share/collectd
#COPY --from=python-plugins /opt/dd/dd-agent /opt/dd/dd-agent
#COPY --from=python-plugins /opt/dd/integrations-core /opt/dd/integrations-core
# Grab pip dependencies too
COPY --from=python-plugins /usr/lib/python2.7/ /usr/lib/python2.7
COPY --from=python-plugins /usr/local/lib/python2.7/ /usr/local/lib/python2.7

# All the built-in collectd plugins
COPY --from=collectd /usr/src/collectd/bindings/java/.libs/*.jar /usr/share/collectd/java/

# Get lib dependencies for collectd and agent
COPY --from=collectd /opt/deps/ /
COPY --from=extra-packages /opt/deps/ /

COPY --from=collectd /usr/lib/jvm/ /usr/lib/jvm
COPY --from=collectd /lib64/ /lib64
COPY --from=collectd /lib/ /lib
COPY --from=collectd /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=collectd /usr/src/collectd/src/types.db /usr/share/collectd/types.db

COPY --from=agent-builder /opt/deps/ /
COPY --from=agent-builder /usr/bin/signalfx-agent /usr/bin/signalfx-agent

COPY neopy /usr/lib/neopy
COPY scripts/agent-status /usr/bin/agent-status

RUN mkdir -p \
      /var/lib/collectd \
	  /var/run \
	  /etc/collectd/managed_config \
	  /etc/collectd/filtering_config &&\
	rm /.dockerenv
RUN chmod +x /usr/bin/signalfx-agent
