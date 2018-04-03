#!/bin/bash

set -e

echo "export PATH="$PATH:/usr/bin"" >> ~/.bashrc

service salt-master start
service salt-minion start

cat <<-EOF >  /srv/salt/top.sls
base:
  '*':
    - signalfx-agent
EOF


cat <<-EOF > /srv/pillar/top.sls
base:
  '*':
    - signalfx-agent
EOF

sleep 60

exec "$@"
