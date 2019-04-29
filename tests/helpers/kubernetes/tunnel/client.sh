#!/bin/bash

set -euo pipefail
set -x

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

context_flag=
if [ -n "${KUBE_CONTEXT-}" ]; then
  context_flag="--context $KUBE_CONTEXT"
fi

out_file=$(mktemp)
function cleanup {
  rm -rf $out_file
  kill $(jobs -p)
}
trap cleanup EXIT

chmod 400 $SCRIPT_DIR/id_rsa

while true; do
  kubectl $context_flag --namespace $NAMESPACE port-forward pod/fake-backend :22 > $out_file &

  for i in seq 1 5; do
    ssh_port=$(cat $out_file | grep -Eo '127.0.0.1:[0-9]+' | sed -e 's/127.0.0.1://' || true)
    if [[ -n "$ssh_port" ]]; then
      break
    fi
    sleep 1
  done

  if [[ -z "$ssh_port" ]]; then
    cat $out_file
    exit 1
  fi

  ssh -N -i $SCRIPT_DIR/id_rsa -p $ssh_port -o 'StrictHostKeyChecking=no' -R 0.0.0.0:$REMOTE_PORT:$LOCAL_HOST:$LOCAL_PORT root@localhost
done
