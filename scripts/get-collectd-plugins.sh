#!/bin/bash

set -exo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

y() {
  filter=$1
  # yq is like jq for yaml
  cat $SCRIPT_DIR/collectd-plugins.yaml | yq -r "$filter"
}

mkdir -p /usr/share/collectd

for i in $(seq 0 $(y '. | length - 1'))
do
  plugin_name=$(y ".[$i].name")
  version=$(y ".[$i].version")
  repo=$(y ".[$i].repo")
  plugin_dir=/usr/share/collectd/${plugin_name}

  git clone --branch $version --depth 1 --single-branch https://github.com/${repo}.git $plugin_dir
  rm -rf $plugin_dir/.git

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
