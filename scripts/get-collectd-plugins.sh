#!/bin/bash

set -exo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TARGET_DIR=${1:-/usr/share/collectd}

y() {
  filter=$1
  # yq is like jq for yaml
  cat $SCRIPT_DIR/../collectd-plugins.yaml | yq -r "$filter"
}

mkdir -p /usr/share/collectd

for i in $(seq 0 $(y '. | length - 1'))
do
  plugin_name=$(y ".[$i].name")
  version=$(y ".[$i].version")
  repo=$(y ".[$i].repo")
  plugin_dir=${TARGET_DIR}/${plugin_name}

  mkdir -p $plugin_dir
  curl -Lo - https://github.com/${repo}/archive/${version}.tar.gz | \
    tar -C $plugin_dir --strip-components=1 -zxf -

  pip_install='pip install'

  if $(y ".[$i] | has(\"pip_packages\")")
  then
    $pip_install $(y ".[$i].pip_packages | join(\" \")")
  elif [[ -f ${plugin_dir}/requirements.txt ]]
  then
    $pip_install -r ${plugin_dir}/requirements.txt
  fi

  if $(y ".[$i] | has(\"can_remove\")")
  then
    for j in $(seq 0 $(y ".[$i].can_remove | length - 1"))
    do
      rm -rf ${plugin_dir}/$(y ".[$i].can_remove[$j]")
    done
  fi
done
