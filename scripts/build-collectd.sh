#!/bin/bash
set -ex -o pipefail

if [ -z "$1" ]; then
  BASE_DIR=`mktemp -d`
else
  # Note: must be absolute path.
  BASE_DIR="$1"
fi

COLLECTD_INCLUDE_DIR="/usr/local/include/collectd"
COLLECTD_LIB_DIR="/usr/local/lib/collectd"
COLLECTD_STATE_DIR="/var"
COLLECTD_SYSCONF_DIR="/etc/collectd"
LIB_DIR="/usr/lib"
PROJECT_DIR=${PROJECT_DIR:-${PWD}}

. ${PROJECT_DIR}/VERSIONS

if [ "$(uname)" == "Darwin" ]; then
  LIB_DIR="/usr/local/lib"
  MS="''"
fi

if [ `id -u` != 0 ]; then
  SUDO=sudo
else
  SUDO=""
fi

mkdir -p ${BASE_DIR}
$SUDO mkdir -p ${COLLECTD_LIB_DIR}
$SUDO mkdir -p ${COLLECTD_INCLUDE_DIR}
$SUDO mkdir -p ${COLLECTD_INCLUDE_DIR}/liboconfig

cd ${BASE_DIR}

[ -e collectd-${COLLECTD_VERSION}.tar.gz ] || curl -OL https://github.com/signalfx/collectd/archive/collectd-${COLLECTD_VERSION}.tar.gz

[ -d collectd-collectd-${COLLECTD_VERSION} ] || tar -xvf collectd-${COLLECTD_VERSION}.tar.gz

cd ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}

cp ${PROJECT_DIR}/collectd-ext/${COLLECTD_VERSION}/plugins/* ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/
cp ${PROJECT_DIR}/collectd-ext/${COLLECTD_VERSION}/daemon/* ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/

[ -e configure ] || ./build.sh
[ -e Makefile ] || ./configure --libdir="${LIB_DIR}" --localstatedir="${COLLECTD_STATE_DIR}" --sysconfdir="${COLLECTD_SYSCONF_DIR}"

make -j4 AM_CFLAGS="-Wall -fPIC -DSIGNALFX_EIM=1" AM_CXXFLAGS="-Wall -fPIC -DSIGNALFX_EIM=1"

$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/*.h ${COLLECTD_INCLUDE_DIR}
$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/liboconfig/*.h ${COLLECTD_INCLUDE_DIR}/liboconfig

$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/daemon/*.o ${COLLECTD_LIB_DIR}
$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/liboconfig/*.o ${COLLECTD_LIB_DIR}

cd ${COLLECTD_LIB_DIR}

$SUDO gcc -shared -o libcollectd.so collectd-collectd.o collectd-meta_data.o collectd-utils_cache.o collectd-utils_llist.o collectd-utils_threshold.o collectd-configfile.o collectd-plugin.o collectd-utils_complain.o collectd-utils_random.o collectd-utils_time.o utils_avltree.o collectd-filter_chain.o collectd-types_list.o collectd-utils_ignorelist.o collectd-utils_subst.o common.o utils_heap.o oconfig.o parser.o scanner.o -ldl -lltdl -lpthread -lm
$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/.libs/python.so ${COLLECTD_LIB_DIR}
$SUDO cp ${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}/src/.libs/aggregation.so ${COLLECTD_LIB_DIR}
