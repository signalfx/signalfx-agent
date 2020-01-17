#!/bin/bash

set -e

echo "export PATH="$PATH:/usr/bin"" >> ~/.bashrc

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

exec "$@"
