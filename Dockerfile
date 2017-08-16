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

ARG collectd_version
RUN cd /tmp &&\
    wget https://github.com/signalfx/collectd/archive/collectd-${collectd_version}.tar.gz &&\
	tar -xvf collectd-${collectd_version}.tar.gz &&\
	mkdir -p /usr/src/ &&\
	mv collectd-collectd* /usr/src/collectd

WORKDIR /usr/src/collectd

ARG extra_cflags="-O2"
ENV CFLAGS "-Wall -fPIC -DSIGNALFX_EIM=1 $extra_cflags"
ENV CXXFLAGS $CFLAGS

RUN ./build.sh &&\
    ./configure \
	  --includedir="/usr/local/include/collectd" \
	  --libdir="/usr/local/lib" \
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
	  --disable-zone

# Overlay our extensions on top of the collectd source
COPY collectd-ext/collectd-sfx/ .

# Compile all of collectd first, including plugins
RUN make -j4

# Make our library version of collectd
RUN gcc -shared -o libcollectd.so \
  src/daemon/collectd-*.o \
  src/daemon/common.o \
  src/daemon/utils_heap.o \
  src/daemon/utils_avltree.o \
  src/liboconfig/*.o \
  -ldl -lltdl -lpthread -lm

# Build our mock of neoagent for testing purposes
COPY collectd-ext/neomock /usr/src/neomock
RUN mkdir -p /usr/local/lib/collectd && cd /usr/src/neomock && make


###### Glide Dependencies Image ######
FROM golang:1.8.3-stretch as godeps

RUN cd /tmp && \
    wget https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz &&\
	tar -xf glide-* &&\
	cp linux-amd64/glide /usr/bin/glide

RUN apt update &&\
    apt install -y git python-pip &&\
	pip install yq &&\
    wget -O /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 &&\
    chmod +x /usr/bin/jq

WORKDIR /go/src/github.com/signalfx/neo-agent
COPY glide.yaml glide.lock ./

# Sed command is a hack to fix a renaming issue with the logrus package
# See https://github.com/sirupsen/logrus/issues/566
RUN glide install --strip-vendor

RUN sed -i -e 's/Sirupsen/sirupsen/' $(grep -lR Sirupsen vendor) &&\
    cp -r vendor/* /go/src/
# Parse glide.lock to get go dep packages and precompile them so later agent
# build is blazing fast
RUN cat glide.lock | tail -n+3 | yq -r '.imports[] | .name' >> /tmp/packages &&\
    cat glide.lock | tail -n+3  | yq -r '.imports[] | select(.subpackages) as $e | .subpackages[] | $e.name + "/" + .' >> /tmp/packages
# A bunch of these fail for some reason, but a lot do compile
RUN for pkg in $(cat /tmp/packages); do go install github.com/signalfx/neo-agent/vendor/$pkg 2>/dev/null; done || true


###### Neoagent Build Image ########
FROM golang:1.8.3-stretch as agent-builder

# Cgo requires dep libraries present to link in libcollectd
RUN apt update &&\
    apt install -y libltdl-dev libzmq5-dev

COPY --from=godeps /go/src/github.com/signalfx/neo-agent/vendor src/github.com/signalfx/neo-agent/vendor
COPY --from=godeps /go/pkg /go/pkg
COPY --from=collectd /usr/src/collectd/ /usr/src/collectd
COPY --from=collectd /usr/src/collectd/libcollectd.so /usr/local/lib/libcollectd.so
ADD scripts/go_packages.tar src/github.com/signalfx/neo-agent/

ARG agent_version
ARG collectd_version
RUN go build \
    -ldflags "-X main.Version=$agent_version -X main.CollectdVersion=$collectd_version -X main.BuiltTime=$(date +%FT%T%z)" \
	-o signalfx-agent \
    github.com/signalfx/neo-agent &&\
	cp signalfx-agent /usr/bin/signalfx-agent


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

RUN bash /opt/get-collectd-plugins.sh

RUN apt install -y libffi-dev libssl-dev build-essential python-dev libcurl4-openssl-dev

COPY scripts/install-dd-plugin-deps.sh /opt/

RUN mkdir -p /opt/dd &&\
    cd /opt/dd &&\
    git clone --depth 1 --single-branch https://github.com/DataDog/dd-agent.git &&\
	git clone --depth 1 --single-branch https://github.com/DataDog/integrations-core.git

RUN bash /opt/install-dd-plugin-deps.sh

COPY neopy/requirements.txt /tmp/requirements.txt
RUN pip install -r /tmp/requirements.txt


###### Final Agent Image #######
FROM ubuntu:16.04 as final-image

ENV DEBIAN_FRONTEND noninteractive
ENV LD_LIBRARY_PATH /usr/lib:/usr/local/lib/collectd:/usr/lib/jvm/java-8-openjdk-amd64/jre/lib/amd64/server

RUN sed -i -e '/^deb-src/d' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y \
      curl \
      debconf \
      default-jre-headless \
      iptables \
      libatasmart4 \
      libc6 \
      libcurl3-gnutls \
      libcurl4-gnutls-dev \
      libdbi1 \
      libesmtp6 \
      libganglia1 \
      libgcrypt20 \
      libglib2.0-0 \
      libldap-2.4-2 \
      libltdl7 \
      liblvm2app2.2 \
      libmemcached11 \
      libmicrohttpd10 \
      libmnl0 \
      libmodbus5 \
      libmysqlclient-dev \
      libmysqlclient20 \
      libnotify4 \
      libopenipmi0 \
      liboping0 \
      libowcapi-3.1-1 \
      libpcap0.8 \
      libperl5.22 \
      libpq5 \
      libprotobuf-c1 \
      libpython2.7 \
      librabbitmq4 \
      librdkafka1 \
      librrd4 \
      libsensors4 \
      libsnmp30 \
      libtokyotyrant3 \
      libudev1 \
      libupsclient4 \
      libvarnishapi1 \
      libvirt0 \
      libxen-4.6 \
      libxml2 \
      libyajl2 \
	  libzmq5 \
	  netcat-openbsd \
      net-tools \
      openjdk-8-jre-headless \
      vim \
      wget

CMD ["/usr/bin/signalfx-agent"]

COPY scripts/debug.sh /opt/debug.sh
ARG DEBUG=false
RUN bash -ec 'if [[ $DEBUG == 'true' ]]; then bash /opt/debug.sh; fi'

LABEL app="signalfx-agent"

RUN mkdir -p /etc/collectd/managed_config /etc/collectd/filtering_config

COPY etc /etc/signalfx/
# Pull in non-C collectd plugins
COPY --from=python-plugins /usr/share/collectd /usr/share/collectd
COPY --from=python-plugins /opt/dd/dd-agent /opt/dd/dd-agent
COPY --from=python-plugins /opt/dd/integrations-core /opt/dd/integrations-core
COPY --from=collectd /usr/src/collectd/src/types.db /usr/share/collectd/types.db
# Grab pip dependencies too
COPY --from=python-plugins /usr/local/lib/python2.7/dist-packages /usr/local/lib/python2.7/dist-packages

COPY --from=collectd /usr/src/collectd/libcollectd.so /usr/local/lib
# All the built-in collectd plugins
COPY --from=collectd /usr/src/collectd/src/.libs/*.so /usr/local/lib/collectd/
COPY --from=collectd /usr/src/collectd/bindings/java/.libs/*.jar /usr/share/collectd/java/
COPY --from=collectd /usr/src/neomock/neomock /usr/bin/neomock

COPY --from=agent-builder /usr/bin/signalfx-agent /usr/bin/signalfx-agent

COPY neopy /usr/local/lib/neopy
COPY scripts/agent-status /usr/bin/agent-status
RUN chmod +x /usr/bin/signalfx-agent
