#!/bin/bash -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHEF_ROOT_DIR="$( cd $SCRIPT_DIR/../ && pwd )"
RUN_OPTS="-d --privileged -v /sys/fs/cgroup:/sys/fs/cgroup:ro -v ${CHEF_ROOT_DIR}:/opt/cookbooks/signalfx_agent:ro -w /opt"
CHEF_CMD="chef-client -z -o 'recipe[signalfx_agent::default]' -j cookbooks/signalfx_agent/test/attributes.json"

for dockerfile in ${SCRIPT_DIR}/Dockerfile.*; do
    distro=$( basename $dockerfile | sed 's/Dockerfile\.//' )
    docker build -t chef-test:$distro -f $dockerfile ${SCRIPT_DIR}
    docker run $RUN_OPTS --name chef-test-$distro chef-test:$distro
    if ! docker exec chef-test-$distro sh -c "$CHEF_CMD"; then
        echo "$distro failed"
        exit 1
    fi
    docker rm -fv chef-test-$distro
done
