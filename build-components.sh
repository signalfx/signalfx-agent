#!/bin/bash
set -xe

if [ -z "$1" ]; then
  BASE_DIR=`mktemp -d`
else
  BASE_DIR="$1"
fi

AGENT_VERSION="1.0.0-beta"
BASE_PACKAGE="github.com/signalfx/neo-agent"
BUILD_TIME=`date +%FT%T%z`
COLLECTD_INCLUDE_DIR="/usr/local/include/collectd"
COLLECTD_LIB_DIR="/usr/local/lib/collectd"
COLLECTD_STATE_DIR="/var"
COLLECTD_SYSCONF_DIR="/etc/collectd"
COLLECTD_VERSION="5.7.0-sfx0"
ENABLE_DEBUG=true
GOPATH="${BASE_DIR}/go"
LIB_DIR="/usr/lib"
MS=""
PROJECT_DIR="${HOME}/work/neo-agent"
PACKAGES=(
     'cmd'
     'plugins'
     'services'
   )

if [ "$(uname)" == "Darwin" ]; then
  LIB_DIR="/usr/local/lib"
  MS="''"
fi

# build collectd shared library
if [ -z $SKIP_LIBCOLLECTD_BUILD ]; then

  mkdir -p ${BASE_DIR}
  mkdir -p ${COLLECTD_LIB_DIR}
  mkdir -p ${COLLECTD_INCLUDE_DIR}
  mkdir -p ${COLLECTD_INCLUDE_DIR}/liboconfig

  cd ${BASE_DIR}

  wget https://github.com/signalfx/collectd/archive/collectd-${COLLECTD_VERSION}.tar.gz

  tar -xvf collectd-${COLLECTD_VERSION}.tar.gz

  cd ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}

  cp ${PROJECT_DIR}/collectd-ext/${COLLECTD_VERSION}/plugins/* ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/
  cp ${PROJECT_DIR}/collectd-ext/${COLLECTD_VERSION}/daemon/* ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/

  ./build.sh
  ./configure --libdir="${LIB_DIR}" --localstatedir="${COLLECTD_STATE_DIR}" --sysconfdir="${COLLECTD_SYSCONF_DIR}"

  make AM_CFLAGS="-Wall -fPIC -DSIGNALFX_EIM=1" AM_CXXFLAGS="-Wall -fPIC -DSIGNALFX_EIM=1"

  cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/*.h ${COLLECTD_INCLUDE_DIR}
  cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/liboconfig/*.h ${COLLECTD_INCLUDE_DIR}/liboconfig

  cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/*.o ${COLLECTD_LIB_DIR}
  cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/liboconfig/*.o ${COLLECTD_LIB_DIR}

  cd ${COLLECTD_LIB_DIR}

  gcc -shared -o libcollectd.so collectd-collectd.o collectd-meta_data.o collectd-utils_cache.o collectd-utils_llist.o collectd-utils_threshold.o collectd-configfile.o collectd-plugin.o collectd-utils_complain.o collectd-utils_random.o collectd-utils_time.o utils_avltree.o collectd-filter_chain.o collectd-types_list.o collectd-utils_ignorelist.o collectd-utils_subst.o common.o utils_heap.o oconfig.o parser.o scanner.o -ldl -lltdl -lpthread -lm

  mkdir -p ${PROJECT_DIR}/.bin
  cp ${COLLECTD_LIB_DIR}/libcollectd.so $PROJECT_DIR/.bin/libcollectd.so
  cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/.libs/python.so $PROJECT_DIR/.bin/python.so

fi

# build agent
if [ -z $SKIP_AGENT_BUILD ]; then

  mkdir -p ${GOPATH}/src/${BASE_PACKAGE}
  mkdir -p ${GOPATH}/pkg
  mkdir -p ${GOPATH}/bin

  for pkg in ${PACKAGES[@]}; do
    cp -r ${PROJECT_DIR}/$pkg ${GOPATH}/src/${BASE_PACKAGE}/
  done

  export GOPATH=$GOPATH

  cd $GOPATH/src/${BASE_PACKAGE}
  go get -d ./...

  cd $GOPATH
  go install -ldflags "-X main.Version=${AGENT_VERSION} -X main.CollectdVersion=${COLLECTD_VERSION} -X main.BuiltTime=${BUILD_TIME}" ${BASE_PACKAGE}/cmd/agent

  mkdir -p ${PROJECT_DIR}/.bin
  cp $GOPATH/bin/agent $PROJECT_DIR/.bin/signalfx-agent
fi

echo "done!"
