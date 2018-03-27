#!/bin/sh

docker exec -it signalfx-agent-salt-test ls /etc/signalfx > signalfx_config.txt
docker exec -it signalfx-agent-salt-test service signalfx-agent status > signalfx_status.txt

grep -q agent.yaml signalfx_config.txt
if [ $? -eq 0 ]
then
    echo "Info: Verified signalfx-agent config"
else
    echo "Error: failed to check signalfx-agent config"
fi

grep -q running signalfx_status.txt
if [ $? -eq 0 ]
then
    echo "Info: Verified the signalfx-agent service"
else
    echo "Error: failed to check signalfx-agent service status"
fi

exit 0