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
ENV CFLAGS "-Wall -fPIC $extra_cflags"
ENV CXXFLAGS $CFLAGS

# In the bundle, the java plugin will live in /plugins/collectd and the JVM
# exists at /jvm
ENV JAVA_LDFLAGS "-Wl,-rpath -Wl,\$\$\ORIGIN/../../jvm/jre/lib/amd64/server"

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

###### Golang Dependencies Image ######
FROM golang:1.9.2-stretch as godeps

RUN wget -O /usr/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 &&\
    chmod +x /usr/bin/dep

WORKDIR /go/src/github.com/signalfx/neo-agent
COPY Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

RUN apt update && apt install -y parallel
# Precompile and cache vendor objects so that building the app is faster
# A bunch of these fail because dep pulls in more than necessary, but a lot do compile
RUN cd vendor && find . -type d -not -empty | grep -v '\btest' | parallel go install {} 2>/dev/null || true


###### Neoagent Build Image ########
FROM ubuntu:16.04 as agent-builder

# Cgo requires dep libraries present
RUN apt update &&\
    apt install -y curl wget pkg-config

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
	mv signalfx-agent /usr/bin/signalfx-agent
	# compressing the binary causes segfaults when run in standalone mode
	#/tmp/upx --lzma /usr/bin/signalfx-agent


###### Python Plugin Image ######
FROM ubuntu:16.04 as python-plugins

RUN apt update &&\
    apt install -y git python-pip wget curl
RUN pip install yq &&\
    wget -O /usr/bin/jq https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 &&\
    chmod +x /usr/bin/jq

# Mirror the same dir structure that exists in the original source
COPY scripts/get-collectd-plugins.sh /opt/scripts/
COPY collectd-plugins.yaml /opt/

RUN mkdir -p /opt/collectd-python &&\
    bash /opt/scripts/get-collectd-plugins.sh /opt/collectd-python

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

####### Extra packages that don't make sense to pull down in any other stage ########
FROM ubuntu:16.04 as extra-packages

RUN apt update &&\
    apt install -y \
	  netcat.openbsd \
	  curl \
	  vim

RUN apt install -y openjdk-8-jre-headless &&\
    cp -rL /usr/lib/jvm/java-8-openjdk-amd64 /opt/jvm &&\
	rm -rf /opt/jvm/docs &&\
	rm -rf /opt/jvm/man

RUN curl -Lo /opt/signalfx_types_db https://raw.githubusercontent.com/signalfx/integrations/master/collectd-java/signalfx_types_db

COPY scripts/collect-libs /opt/collect-libs
RUN /opt/collect-libs /opt/deps /usr/bin/vim /usr/bin/curl /bin

###### Final Agent Image #######
# This build stage is meant as the final target when running the agent in a
# container environment (e.g. directly with Docker or on K8s).  The stages
# below this are special-purpose.
FROM scratch as final-image

CMD ["/usr/bin/signalfx-agent"]

# Pull in non-C collectd plugins
COPY --from=python-plugins /opt/collectd-python/ /plugins/collectd
#COPY --from=python-plugins /opt/dd/dd-agent /opt/dd/dd-agent
#COPY --from=python-plugins /opt/dd/integrations-core /opt/dd/integrations-core
# Grab pip dependencies too
COPY --from=python-plugins /usr/lib/python2.7/ /lib/python2.7
COPY --from=python-plugins /usr/local/lib/python2.7/ /lib/python2.7

COPY --from=extra-packages /lib/x86_64-linux-gnu/ld-2.23.so /lib64/ld-linux-x86-64.so.2
COPY --from=extra-packages /opt/deps/ /lib
COPY --from=extra-packages /usr/bin/vim /bin/vim
COPY --from=extra-packages /usr/bin/curl /bin/curl
COPY --from=extra-packages /bin/ /bin
COPY --from=extra-packages /opt/jvm/ /jvm
COPY --from=extra-packages /opt/signalfx_types_db /plugins/collectd/java/

RUN mkdir /run

COPY scripts/agent-status /bin/agent-status

COPY --from=collectd /usr/sbin/collectd /bin/collectd
COPY --from=collectd /opt/deps/ /lib

COPY --from=collectd /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=collectd /usr/share/collectd/types.db /plugins/collectd/types.db
# All the built-in collectd plugins
COPY --from=collectd /usr/lib/collectd/*.so /plugins/collectd/
COPY --from=collectd /usr/share/collectd/java/ /plugins/collectd/java/

COPY neopy /neopy
COPY --from=agent-builder /usr/bin/signalfx-agent /bin/signalfx-agent

# The current directory of the agent is important since it uses a lot of
# relative paths to make it very easily relocated within the filesystem in
# standalone.
WORKDIR /

###### Standalone Image Filesystem ########
# This is an image almost identical to the normal one above except that it has
# the runc binary and config for running the agent apart from docker (but still
# using containers).  This image is meant to be dumped to a tar file and run
# standalone.
FROM scratch as standalone

COPY --from=final-image / /bundle

COPY scripts/standalone/run-agent /run-agent


####### Dev Image ########
# This is an image to facilitate development of the agent.  It installs all of
# the build tools for building collectd and the go agent, along with some other
# useful utilities.  The agent image is copied from the final-image stage to
# the /agent dir in here, and should be run with a chroot jail to closely
# mimick the way the agent is normally run.  There are targets in the Makefile
# to assist with running it in the chroot jail.
FROM ubuntu:16.04 as dev-extras

RUN apt update &&\
    apt install -y libcurl4-openssl-dev &&\
    apt install -y \
      autoconf \
      automake \
      autotools-dev \
      bison \
      build-essential \
      flex \
      git \
      inotify-tools \
      libtool \
      pkg-config \
      python-pip \
	  vim \
	  curl \
	  wget

RUN pip install ipython==5 ipdb

WORKDIR /go/src/github.com/signalfx/neo-agent
CMD ["/bin/bash"]
ENV PATH=$PATH:/usr/local/go/bin:/go/bin GOPATH=/go

COPY --from=agent-builder /usr/local/go/ /usr/local/go
COPY --from=godeps /usr/bin/dep /usr/bin/dep
COPY --from=collectd /usr/src/collectd/ /usr/src/collectd

RUN go get -u github.com/golang/lint/golint
RUN go get github.com/derekparker/delve/cmd/dlv &&\
    cp /go/bin/dlv /agent/usr/bin/dlv

COPY --from=godeps /go/src/github.com/signalfx/neo-agent/vendor src/github.com/signalfx/neo-agent/vendor
COPY --from=godeps /go/pkg /go/pkg

COPY --from=final-image / /bundle/
COPY scripts/standalone/run-agent /
