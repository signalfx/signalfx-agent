#!/bin/bash

set -eo pipefail

[ -n "$DEPLOYMENT_TYPE" ] || (echo "DEPLOYMENT_TYPE not defined!" && exit 1)

mkdir -p ~/testresults
if [ "$CIRCLE_BRANCH" != "master" ]; then
    if ! scripts/changes-include-dir deployments/${DEPLOYMENT_TYPE} tests/deployments/${DEPLOYMENT_TYPE} ${BASH_SOURCE[0]}; then
        echo "${DEPLOYMENT_TYPE} code has not changed, skipping tests!"
        touch ~/.skip
        exit 0
    fi
fi

if [ ! -d tests/deployments/${DEPLOYMENT_TYPE} ]; then
    # no pytest tests to execute for deployment
    touch ~/.skip
    # only run subsequent steps on node 0
    [ $CIRCLE_NODE_INDEX -eq 0 ] || exit 0
fi

cd deployments/${DEPLOYMENT_TYPE}
make dev-image

case "${DEPLOYMENT_TYPE}" in
chef)
    CHEF_DEV="docker run --rm \
        --workdir /chef-repo/cookbooks/signalfx_agent \
        signalfx-agent-chef-dev"

    $CHEF_DEV chef exec rspec --format RspecJunitFormatter | sed '/No examples found./d' | tee ~/testresults/chefspec.xml
    $CHEF_DEV foodcritic .
    $CHEF_DEV cookstyle .
    if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
        echo "export MARKERS='chef and rpm'" >> $BASH_ENV
    else
        echo "export MARKERS='chef and deb'" >> $BASH_ENV
    fi
    ;;
puppet)
    docker run --rm \
        -v ~/testresults:/testresults \
        -e "CI_SPEC_OPTIONS=--format RspecJunitFormatter -o /testresults/puppetspec.xml" \
        signalfx-agent-puppet-dev \
        rake spec
    docker run --rm \
        signalfx-agent-puppet-dev \
        puppet-lint --fail-on-warnings .
    if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
        echo "export MARKERS='puppet and rpm'" >> $BASH_ENV
    else
        echo "export MARKERS='puppet and deb'" >> $BASH_ENV
    fi
    ;;
salt)
    docker run --rm \
        signalfx-agent-salt-dev \
        salt '*' state.apply | tee ~/testresults/salt.out
    ;;
ansible)
    docker run --rm \
        --cap-add DAC_READ_SEARCH \
        --cap-add SYS_PTRACE \
        signalfx-agent-ansible-dev \
        ansible-playbook -i inventory example-playbook.yml --connection=local \
        -e '{"sfx_agent_config": {"signalFxAccessToken": "MyToken"}}' | tee ~/testresults/ansible.out
    docker run --rm \
        signalfx-agent-ansible-dev \
        ansible-lint .
    if [ $CIRCLE_NODE_INDEX -eq 0 ]; then
        echo "export MARKERS='ansible and rpm'" >> $BASH_ENV
    else
        echo "export MARKERS='ansible and deb'" >> $BASH_ENV
    fi
    ;;
*)
    echo "Deployment ${DEPLOYMENT_TYPE} not supported!"
    exit 1
esac
