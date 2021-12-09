#!/bin/bash

set -eo pipefail

[ -n "$DEPLOYMENT_TYPE" ] || (echo "DEPLOYMENT_TYPE not defined!" && exit 1)

mkdir -p "$HOME/$RESULT_PATH"

if [ ! -d tests/deployments/$DEPLOYMENT_TYPE ]; then
    # no pytest tests to execute for deployment
    touch ~/.skip
    # only run subsequent steps for rpm
    [ $SYS_PACKAGE = "rpm" ] || exit 0
fi

cd deployments/${DEPLOYMENT_TYPE}
make dev-image

case "${DEPLOYMENT_TYPE}" in
chef)
    CHEF_DEV="docker run --rm \
        --workdir /chef-repo/cookbooks/signalfx_agent \
        signalfx-agent-chef-dev"
    $CHEF_DEV chef exec rspec --format RspecJunitFormatter | sed '/No examples found./d' | tee $HOME/$RESULT_PATH/chefspec.xml
    $CHEF_DEV foodcritic .
    $CHEF_DEV cookstyle .
    if [ $SYS_PACKAGE = "rpm" ]; then
        echo 'MARKERS=chef and rpm' >> $GITHUB_ENV
    else
        echo 'MARKERS=chef and deb' >> $GITHUB_ENV
    fi
    ;;
puppet)
    docker run --rm \
        -v $HOME/$RESULT_PATH:/$RESULT_PATH \
        -e "CI_SPEC_OPTIONS=--format RspecJunitFormatter -o /$RESULT_PATH/puppetspec.xml" \
        signalfx-agent-puppet-dev \
        rake spec
    docker run --rm \
        signalfx-agent-puppet-dev \
        puppet-lint --fail-on-warnings .
    if [ $SYS_PACKAGE = "rpm" ]; then
        echo "MARKERS=puppet and rpm" >> $GITHUB_ENV
    else
        echo "MARKERS=puppet and deb" >> $GITHUB_ENV
    fi
    ;;
salt)
    docker run --rm signalfx-agent-salt-dev \
        make -f /Makefile test 2>&1 | tee $HOME/$RESULT_PATH/salt.out
    if [ $SYS_PACKAGE = "rpm" ]; then
        echo "MARKERS=salt and rpm" >> $GITHUB_ENV
    else
        echo "MARKERS=salt and deb" >> $GITHUB_ENV
    fi
    ;;
ansible)
    docker run --rm \
        --cap-add DAC_READ_SEARCH \
        --cap-add SYS_PTRACE \
        signalfx-agent-ansible-dev \
        ansible-playbook -i inventory example-playbook.yml --connection=local \
        -e '{"sfx_agent_config": {"signalFxAccessToken": "MyToken"}}' | tee $HOME/$RESULT_PATH/ansible.out
    docker run --rm \
        signalfx-agent-ansible-dev \
        ansible-lint -x experimental roles/signalfx-agent
    if [ $SYS_PACKAGE = "rpm" ]; then
        echo "MARKERS=ansible and rpm" >> $GITHUB_ENV
    else
        echo "MARKERS=ansible and deb" >> $GITHUB_ENV
    fi
    ;;
*)
    echo "Deployment ${DEPLOYMENT_TYPE} not supported!"
    exit 1
esac
