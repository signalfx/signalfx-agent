#!/bin/bash -e

mount --make-shared /
dockerd \
  --host=unix:///var/run/docker.sock \
  --host=tcp://0.0.0.0:2375 \
  --insecure-registry localhost:5000 \
  --metrics-addr 127.0.0.1:9323 \
  --experimental \
  &> /var/log/docker.log 2>&1 < /dev/null &

TIMEOUT=30
START_TIME=`date +%s`
while [ 0 ]; do
    if [ $(expr `date +%s` - $START_TIME) -gt $TIMEOUT ]; then
        exit 1
    fi
    if docker version >/dev/null 2>&1; then
        break
    fi
    sleep 2
done

docker version
