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

src_dir=${BASE_DIR}/collectd-collectd-${COLLECTD_VERSION}
cd $src_dir

cp -r ${PROJECT_DIR}/collectd-ext/collectd-sfx/* $src_dir

[ -e configure ] || ./build.sh

CFLAGS="-Wall -fPIC -DSIGNALFX_EIM=1"

if [[ $DEBUG == "true" ]]
then
  CFLAGS="$CFLAGS -g -O0"
else
  CFLAGS="$CFLAGS -O2"
fi

export CFLAGS
export CXXFLAGS=$CFLAGS

[ -e Makefile ] || ./configure --libdir="${LIB_DIR}" --localstatedir="${COLLECTD_STATE_DIR}" --sysconfdir="${COLLECTD_SYSCONF_DIR}"

make -j4

$SUDO cp ${src_dir}/src/daemon/*.h ${COLLECTD_INCLUDE_DIR}
$SUDO cp ${src_dir}/src/liboconfig/*.h ${COLLECTD_INCLUDE_DIR}/liboconfig

$SUDO cp ${src_dir}/src/daemon/*.o ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/liboconfig/*.o ${COLLECTD_LIB_DIR}

cd ${COLLECTD_LIB_DIR}

$SUDO gcc -shared -o libcollectd.so collectd-collectd.o collectd-meta_data.o collectd-utils_cache.o collectd-utils_llist.o collectd-utils_threshold.o collectd-configfile.o collectd-plugin.o collectd-utils_complain.o collectd-utils_random.o collectd-utils_time.o utils_avltree.o collectd-filter_chain.o collectd-types_list.o collectd-utils_ignorelist.o collectd-utils_subst.o common.o utils_heap.o oconfig.o parser.o scanner.o -ldl -lltdl -lpthread -lm
$SUDO cp ${src_dir}/src/.libs/java.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/.libs/memcached.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/.libs/mysql.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/.libs/nginx.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/.libs/python.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/src/.libs/aggregation.so ${COLLECTD_LIB_DIR}
$SUDO cp ${src_dir}/bindings/java/.libs/generic-jmx.jar ${COLLECTD_LIB_DIR}

# This will remain empty if DEBUG is false
mkdir -p /opt/collectd-src
if [[ $DEBUG == "true" ]]
then
  cp -r ${src_dir} /opt/collectd-src
fi
