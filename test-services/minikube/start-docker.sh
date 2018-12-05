#!/bin/bash -e

if docker version >/dev/null 2>&1; then
    exit 0
fi

DOCKER_OPTS="--host=unix:///var/run/docker.sock --host=tcp://0.0.0.0:2375 --insecure-registry localhost:5000 --metrics-addr 127.0.0.1:9323 --experimental"

if pgrep systemd >/dev/null 2>&1; then
    sed -i "s|^ExecStart=/usr/bin/dockerd .*|ExecStart=/usr/bin/dockerd ${DOCKER_OPTS}|" /lib/systemd/system/docker.service
    systemctl enable docker.service
    systemctl start docker.service
else
    /usr/bin/dockerd $DOCKER_OPTS &> /var/log/docker.log 2>&1 < /dev/null &
fi

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
